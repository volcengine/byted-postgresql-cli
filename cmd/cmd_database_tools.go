// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cli

import (
	"bytes"
	stdsql "database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

// databaseTarget contains the two ways the PostgreSQL data-plane commands can
// obtain a libpq connection: a caller supplied URL or a Volcengine AIDAP
// workspace/branch resolved to a role connection string.
type databaseTarget struct {
	url         string
	workspaceID string
	branchID    string
	database    string
	role        string
}

func newDatabaseToolsCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Run PostgreSQL queries and schema tools",
	}
	cmd.AddCommand(
		newDBQueryCmd(),
		newDBDumpCmd(),
		newDBPullCmd(),
		newDBAdvisorsCmd(),
	)
	return cmd
}

func newDatabaseInspectCmd(ctx ProviderContext) *cobra.Command {
	var target databaseTarget
	var unaligned bool
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect PostgreSQL database health and usage",
	}
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Inspect a PostgreSQL database",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown inspect db command %q", args[0])
			}
			return cmd.Help()
		},
	}
	queries := map[string]struct {
		short string
		sql   string
	}{
		"db-stats": {
			"Show database size and cache hit rates",
			`SELECT current_database() AS database,
	pg_size_pretty(pg_database_size(current_database())) AS database_size,
	round(sum(blks_hit)::numeric / NULLIF(sum(blks_hit + blks_read), 0) * 100, 2) AS cache_hit_rate
FROM pg_stat_database
WHERE datname = current_database();`,
		},
		"index-stats": {
			"Show index sizes and scan counts",
			`SELECT schemaname, relname AS table_name, indexrelname AS index_name,
	pg_size_pretty(pg_relation_size(indexrelid)) AS index_size, idx_scan
FROM pg_stat_user_indexes ORDER BY pg_relation_size(indexrelid) DESC;`,
		},
		"table-stats": {
			"Show table sizes and estimated row counts",
			`SELECT schemaname, relname AS table_name, n_live_tup, n_dead_tup,
	pg_size_pretty(pg_total_relation_size(relid)) AS total_size
FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC;`,
		},
		"role-stats": {
			"Show PostgreSQL roles and connection limits",
			`SELECT rolname, rolsuper, rolcreaterole, rolcreatedb, rolcanlogin,
	rolconnlimit FROM pg_roles ORDER BY rolname;`,
		},
		"long-running-queries": {
			"Show queries running longer than five minutes",
			`SELECT pid, usename, datname, now() - query_start AS duration, state, query
FROM pg_stat_activity WHERE query_start IS NOT NULL
	AND now() - query_start > interval '5 minutes'
ORDER BY query_start;`,
		},
		"locks": {
			"Show active PostgreSQL locks",
			`SELECT pid, locktype, mode, relation::regclass AS relation, granted
FROM pg_locks WHERE NOT granted ORDER BY pid;`,
		},
		"blocking": {
			"Show blocked and blocking sessions",
			`SELECT blocked.pid AS blocked_pid, blocking.pid AS blocking_pid,
	blocked.query AS blocked_query, blocking.query AS blocking_query
FROM pg_stat_activity blocked
JOIN pg_locks blocked_lock ON blocked.pid = blocked_lock.pid AND NOT blocked_lock.granted
JOIN pg_locks blocking_lock ON blocking_lock.locktype = blocked_lock.locktype
	AND blocking_lock.database IS NOT DISTINCT FROM blocked_lock.database
	AND blocking_lock.relation IS NOT DISTINCT FROM blocked_lock.relation
	AND blocking_lock.granted
JOIN pg_stat_activity blocking ON blocking.pid = blocking_lock.pid
WHERE blocked.pid <> blocking.pid;`,
		},
		"replication-slots": {
			"Show replication slots",
			`SELECT slot_name, plugin, slot_type, database, active, restart_lsn,
	confirmed_flush_lsn FROM pg_replication_slots ORDER BY slot_name;`,
		},
		"vacuum-stats": {
			"Show vacuum and analyze statistics",
			`SELECT schemaname, relname AS table_name, last_vacuum, last_autovacuum,
	last_analyze, last_autoanalyze, vacuum_count, autovacuum_count
FROM pg_stat_user_tables ORDER BY relname;`,
		},
		"traffic-profile": {
			"Show table read and write activity",
			`SELECT schemaname, relname AS table_name, seq_scan, idx_scan,
	n_tup_ins, n_tup_upd, n_tup_del FROM pg_stat_user_tables
ORDER BY seq_scan + idx_scan DESC;`,
		},
		"outliers": {
			"Show slow queries from pg_stat_statements",
			`SELECT query, calls, total_exec_time, mean_exec_time, rows
FROM pg_stat_statements ORDER BY total_exec_time DESC LIMIT 100;`,
		},
		"calls": {
			"Show most frequently called queries",
			`SELECT query, calls, total_exec_time, mean_exec_time, rows
FROM pg_stat_statements ORDER BY calls DESC LIMIT 100;`,
		},
		"bloat": {
			"Show tables with dead tuples",
			`SELECT schemaname, relname AS table_name, n_live_tup, n_dead_tup,
	round(n_dead_tup::numeric / NULLIF(n_live_tup + n_dead_tup, 0) * 100, 2) AS dead_tuple_pct
FROM pg_stat_user_tables WHERE n_dead_tup > 0
ORDER BY n_dead_tup DESC;`,
		},
	}
	for name, spec := range queries {
		name, spec := name, spec
		child := &cobra.Command{
			Use:   name,
			Short: spec.short,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				format, err := sqlOutputFormat(cmd, unaligned)
				if err != nil {
					return err
				}
				return runPSQLTarget(cmd, target, spec.sql, format)
			},
		}
		dbCmd.AddCommand(child)
	}
	addDatabaseTargetFlags(dbCmd, &target)
	dbCmd.PersistentFlags().BoolVar(&unaligned, "unaligned", false, "Print rows psql-style, pipe-separated (cannot combine with --output/-o)")
	cmd.AddCommand(dbCmd)
	return cmd
}

func newDBQueryCmd() *cobra.Command {
	var target databaseTarget
	var file string
	var unaligned bool
	cmd := &cobra.Command{
		Use:   "query [sql]",
		Short: "Execute a SQL query against PostgreSQL",
		Long: `Execute a SQL query against PostgreSQL.

Results honor the global --output/-o flag (table, json, yaml, csv, or tsv), so
db query behaves like every other command. Use --unaligned for psql-style,
pipe-separated rows; it cannot be combined with an explicit --output.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sql, err := queryText(args, file)
			if err != nil {
				return err
			}
			g := fromCtx(cmd)
			format := g.Output
			if unaligned {
				if cmd.Flags().Changed("output") {
					return fmt.Errorf("--unaligned cannot be combined with --output/-o")
				}
				format = "unaligned"
			}
			return runPSQLTarget(cmd, target, sql, format)
		},
	}
	addDatabaseTargetFlags(cmd, &target)
	cmd.Flags().StringVarP(&file, "file", "f", "", "Read SQL from a file")
	cmd.Flags().BoolVar(&unaligned, "unaligned", false, "Print rows psql-style, pipe-separated (cannot combine with --output/-o)")
	return cmd
}

func newDBDumpCmd() *cobra.Command {
	var target databaseTarget
	var file string
	var dataOnly, schemaOnly, rolesOnly, dryRun, reveal, masked bool
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump a PostgreSQL database with pg_dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun && (reveal || masked) {
				return fmt.Errorf("--reveal and --masked can only be used with --dry-run")
			}
			url, err := resolveDatabaseURL(cmd, target)
			if err != nil {
				return err
			}
			if dataOnly && schemaOnly {
				return fmt.Errorf("--data-only and --schema-only are mutually exclusive")
			}
			bin, err := pgDumpBinary(dryRun)
			if err != nil {
				return err
			}
			if target.workspaceID != "" && bin != "pg_dump" {
				client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
				if err != nil {
					return err
				}
				if err := checkPostgresClientVersion(cmd, client, target.workspaceID, bin, "pg_dump"); err != nil {
					return err
				}
			}
			pgArgs := []string{"--no-password", "--dbname", url}
			switch {
			case rolesOnly:
				pgArgs = append(pgArgs, "--roles-only")
			case dataOnly:
				pgArgs = append(pgArgs, "--data-only")
			case schemaOnly:
				pgArgs = append(pgArgs, "--schema-only")
			}
			if file != "" {
				pgArgs = append(pgArgs, "--file", file)
			}
			if dryRun {
				if reveal && masked {
					return fmt.Errorf("--reveal cannot be combined with --masked")
				}
				displayArgs := pgArgs
				if !reveal {
					fmt.Fprintln(cmd.ErrOrStderr(), "Notice: dry-run output is a command template with the password masked as *****; replace it with the real password before executing.")
					displayArgs = append([]string(nil), pgArgs...)
					displayArgs[2] = maskConnectionPassword(displayArgs[2])
				} else {
					fmt.Fprintln(cmd.ErrOrStderr(), "Warning: dry-run output contains a password; protect it like a secret.")
				}
				formattedArgs := make([]string, len(displayArgs))
				for i, arg := range displayArgs {
					formattedArgs[i] = shellQuote(arg)
				}
				fmt.Fprintln(cmd.OutOrStdout(), shellQuote(bin)+" "+strings.Join(formattedArgs, " "))
				return nil
			}
			return runExternal(cmd, bin, pgArgs...)
		},
	}
	addDatabaseTargetFlags(cmd, &target)
	cmd.Flags().StringVarP(&file, "file", "f", "", "Write the dump to a file")
	cmd.Flags().BoolVar(&dataOnly, "data-only", false, "Dump data only")
	cmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Dump schema only")
	cmd.Flags().BoolVar(&rolesOnly, "roles-only", false, "Dump roles only")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the pg_dump command without executing it")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "With --dry-run, print an executable command with the password in plain text (sensitive)")
	cmd.Flags().BoolVar(&masked, "masked", false, "With --dry-run, mask the password in the command output (default)")
	cmd.MarkFlagsMutuallyExclusive("data-only", "schema-only", "roles-only")
	return cmd
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	for _, character := range value {
		if strings.ContainsRune(" \t\n\r\"'\\$&;|<>`(){}[]*?!#", character) {
			return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
		}
	}
	return value
}

func pgDumpBinary(dryRun bool) (string, error) {
	bin, err := exec.LookPath("pg_dump")
	if err == nil {
		return bin, nil
	}
	if dryRun {
		return "pg_dump", nil
	}
	return "", fmt.Errorf("pg_dump not found in PATH: %w", err)
}

func newDBPullCmd() *cobra.Command {
	var target databaseTarget
	var file string
	cmd := &cobra.Command{
		Use:   "pull [file]",
		Short: "Pull the PostgreSQL schema into a SQL file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" && len(args) == 1 {
				file = args[0]
			}
			url, err := resolveDatabaseURL(cmd, target)
			if err != nil {
				return err
			}
			bin, err := exec.LookPath("pg_dump")
			if err != nil {
				return fmt.Errorf("pg_dump not found in PATH: %w", err)
			}
			if target.workspaceID != "" {
				client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
				if err != nil {
					return err
				}
				if err := checkPostgresClientVersion(cmd, client, target.workspaceID, bin, "pg_dump"); err != nil {
					return err
				}
			}
			pgArgs := []string{"--no-password", "--schema-only", "--dbname", url}
			if file != "" {
				pgArgs = append(pgArgs, "--file", file)
			}
			return runExternal(cmd, bin, pgArgs...)
		},
	}
	addDatabaseTargetFlags(cmd, &target)
	cmd.Flags().StringVarP(&file, "file", "f", "", "Write the schema to a file")
	return cmd
}

func newDBAdvisorsCmd() *cobra.Command {
	var target databaseTarget
	var unaligned bool
	cmd := &cobra.Command{
		Use:   "advisors",
		Short: "Check PostgreSQL for common security and performance issues",
		Long: `Run independent PostgreSQL health checks and report actionable findings.

Checks include dead tuples, missing primary keys, unused indexes, long
transactions, and slow queries when pg_stat_statements is available.
Unavailable optional checks are reported as skipped findings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := sqlOutputFormat(cmd, unaligned)
			if err != nil {
				return err
			}
			return runDatabaseAdvisors(cmd, target, format)
		},
	}
	addDatabaseTargetFlags(cmd, &target)
	cmd.Flags().BoolVar(&unaligned, "unaligned", false, "Print rows psql-style, pipe-separated (cannot combine with --output/-o)")
	return cmd
}

func addDatabaseTargetFlags(cmd *cobra.Command, target *databaseTarget) {
	flags := cmd.PersistentFlags()
	flags.StringVar(&target.url, "db-url", "", "PostgreSQL connection URL")
	flags.StringVar(&target.workspaceID, "workspace-id", "", "PostgreSQL workspace ID")
	flags.StringVar(&target.branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
	flags.StringVar(&target.database, "database-name", "", "Database name")
	flags.StringVar(&target.role, "role-name", "", "PostgreSQL role/account name")
	cmd.MarkFlagsMutuallyExclusive("db-url", "workspace-id")
}

func sqlOutputFormat(cmd *cobra.Command, unaligned bool) (string, error) {
	if unaligned {
		if cmd.Flags().Changed("output") {
			return "", fmt.Errorf("--unaligned cannot be combined with --output/-o")
		}
		return "unaligned", nil
	}
	return fromCtx(cmd).Output, nil
}

func queryText(args []string, file string) (string, error) {
	if file != "" {
		body, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read SQL file: %w", err)
		}
		return string(body), nil
	}
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return "", fmt.Errorf("SQL is required; pass it as an argument or with --file")
	}
	return args[0], nil
}

func resolveDatabaseURL(cmd *cobra.Command, target databaseTarget) (string, error) {
	if strings.TrimSpace(target.url) != "" {
		return psqlConnectionURL(volcengine.DBAccountConnection{ConnectionURL: target.url})
	}
	if target.workspaceID == "" {
		return "", fmt.Errorf("no database target selected\n\nRun:\n  byted-postgresql-cli workspaces list\n\nThen retry with:\n  --db-url <url> or --workspace-id <workspace-id>")
	}
	g := fromCtx(cmd)
	client, err := g.NewVolcClient(cmd.Context())
	if err != nil {
		return "", err
	}
	conn, err := resolveConnection(cmd.Context(), client, target.workspaceID, target.branchID, target.database, target.role)
	if err != nil {
		return "", err
	}
	return psqlConnectionURL(conn)
}

func runPSQLTarget(cmd *cobra.Command, target databaseTarget, sql, format string) error {
	connectionURL, err := resolveDatabaseTarget(cmd, target)
	if err != nil {
		return err
	}
	db, err := stdsql.Open("pgx", connectionURL)
	if err != nil {
		return fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(cmd.Context()); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	rows, err := db.QueryContext(cmd.Context(), sql)
	if err != nil {
		return fmt.Errorf("execute PostgreSQL query: %w", translatePostgreSQLQueryError(err))
	}
	defer rows.Close()
	results, err := collectSQLResults(rows)
	if err != nil {
		return err
	}
	return writeSQLResults(cmd, results, format)
}

type sqlResult struct {
	columns []string
	values  [][]any
}

func collectSQLResults(rows *stdsql.Rows) ([]sqlResult, error) {
	var results []sqlResult
	for {
		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		result := sqlResult{columns: columns}
		for rows.Next() {
			row := make([]any, len(columns))
			dest := make([]any, len(columns))
			for i := range row {
				dest[i] = &row[i]
			}
			if err := rows.Scan(dest...); err != nil {
				return nil, err
			}
			result.values = append(result.values, row)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		results = append(results, result)
		if !rows.NextResultSet() {
			break
		}
	}
	return results, nil
}

func writeSQLResults(cmd *cobra.Command, results []sqlResult, format string) error {
	if len(results) == 1 {
		return writeSQLResult(cmd, results[0].columns, results[0].values, format)
	}
	out := cmd.OutOrStdout()
	if format == "json" || format == "yaml" {
		payload := make([]map[string]any, 0, len(results))
		for _, result := range results {
			payload = append(payload, map[string]any{
				"columns": result.columns,
				"rows":    sqlRowsAsObjects(result.columns, result.values),
			})
		}
		if format == "json" {
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(payload)
		}
		return yaml.NewEncoder(out).Encode(payload)
	}
	for i, result := range results {
		if i > 0 && format == "table" {
			fmt.Fprintln(out)
		}
		if err := writeSQLResult(cmd, result.columns, result.values, format); err != nil {
			return err
		}
	}
	return nil
}

func sqlRowsAsObjects(columns []string, values [][]any) []map[string]any {
	rows := make([]map[string]any, 0, len(values))
	for _, valuesRow := range values {
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			if i < len(valuesRow) {
				row[column] = normalizeSQLValue(valuesRow[i])
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func translatePostgreSQLQueryError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == "42P01" &&
		strings.Contains(pgErr.Message, `relation "pg_stat_statements" does not exist`) {
		return errors.New("the pg_stat_statements extension is not enabled; run CREATE EXTENSION pg_stat_statements in postgres or ask a DBA to enable it")
	}
	return err
}

func resolveDatabaseTarget(cmd *cobra.Command, target databaseTarget) (connectionURL string, err error) {
	if strings.TrimSpace(target.url) != "" {
		return psqlConnectionURL(volcengine.DBAccountConnection{ConnectionURL: target.url})
	}
	if target.workspaceID == "" {
		return "", fmt.Errorf("no database target selected\n\nRun:\n  byted-postgresql-cli workspaces list\n\nThen retry with:\n  --db-url <url> or --workspace-id <workspace-id>")
	}
	g := fromCtx(cmd)
	client, err := g.NewVolcClient(cmd.Context())
	if err != nil {
		return "", err
	}
	conn, err := resolveConnection(cmd.Context(), client, target.workspaceID, target.branchID, target.database, target.role)
	if err != nil {
		return "", err
	}
	return psqlConnectionURL(conn)
}

// writeSQLResult renders a query result set in the requested global output
// format. It is deliberately self-contained rather than routed through the
// shared writer package: that writer snake_cases keys/headers for Go struct
// fields, which would silently rewrite arbitrary SQL column aliases. Here the
// exact column names and order are preserved across every format.
func writeSQLResult(cmd *cobra.Command, columns []string, values [][]any, format string) error {
	out := cmd.OutOrStdout()
	switch format {
	case "unaligned":
		return writeSQLUnaligned(out, values)
	case "csv":
		return writeSQLSeparated(out, columns, values, ",")
	case "json":
		return writeSQLJSON(out, columns, values)
	case "yaml":
		return writeSQLYAML(out, columns, values)
	case "table":
		return writeSQLTable(out, columns, values)
	default:
		// Keep tsv as a raw, machine-readable tab-separated format.
		return writeSQLSeparated(out, columns, values, "\t")
	}
}

// writeSQLUnaligned prints psql-style, pipe-separated rows without a header.
func writeSQLUnaligned(out io.Writer, values [][]any) error {
	for _, row := range values {
		for i, value := range row {
			if i > 0 {
				fmt.Fprint(out, "|")
			}
			fmt.Fprint(out, value)
		}
		fmt.Fprintln(out)
	}
	return nil
}

// writeSQLSeparated prints a header row plus data rows joined by sep.
func writeSQLSeparated(out io.Writer, columns []string, values [][]any, sep string) error {
	fmt.Fprintln(out, strings.Join(columns, sep))
	for _, row := range values {
		cells := make([]string, len(row))
		for i, value := range row {
			cells[i] = fmt.Sprint(value)
		}
		fmt.Fprintln(out, strings.Join(cells, sep))
	}
	return nil
}

// writeSQLTable renders SQL results as an aligned, borderless ASCII table.
// SQL column names are preserved exactly as returned by PostgreSQL, including
// aliases and their order.
func writeSQLTable(out io.Writer, columns []string, values [][]any) error {
	table := tablewriter.NewWriter(out)
	table.SetHeader(columns)
	table.SetAutoWrapText(false)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator(" ")
	for _, row := range values {
		cells := make([]string, len(row))
		for i, value := range row {
			cells[i] = fmt.Sprint(normalizeSQLValue(value))
		}
		table.Append(cells)
	}
	table.Render()
	return nil
}

// writeSQLJSON emits the result as an array of objects, one per row, with
// column order preserved (encoding/json would otherwise sort map keys).
func writeSQLJSON(out io.Writer, columns []string, values [][]any) error {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for r, row := range values {
		if r > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		for c, col := range columns {
			if c > 0 {
				buf.WriteByte(',')
			}
			key, err := json.Marshal(col)
			if err != nil {
				return err
			}
			buf.Write(key)
			buf.WriteByte(':')
			val, err := json.Marshal(normalizeSQLValue(row[c]))
			if err != nil {
				return err
			}
			buf.Write(val)
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, buf.Bytes(), "", "  "); err != nil {
		return err
	}
	pretty.WriteByte('\n')
	_, err := out.Write(pretty.Bytes())
	return err
}

// writeSQLYAML emits the result as a sequence of mappings, preserving column
// order via explicit yaml nodes.
func writeSQLYAML(out io.Writer, columns []string, values [][]any) error {
	root := &yaml.Node{Kind: yaml.SequenceNode}
	for _, row := range values {
		mapping := &yaml.Node{Kind: yaml.MappingNode}
		for c, col := range columns {
			keyNode := &yaml.Node{}
			keyNode.SetString(col)
			valNode := &yaml.Node{}
			if err := valNode.Encode(normalizeSQLValue(row[c])); err != nil {
				return err
			}
			mapping.Content = append(mapping.Content, keyNode, valNode)
		}
		root.Content = append(root.Content, mapping)
	}
	enc := yaml.NewEncoder(out)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return err
	}
	return enc.Close()
}

// normalizeSQLValue makes a scanned driver value safe for structured encoding:
// []byte (text/bytea columns) is rendered as a string rather than a base64
// blob / byte list, while other types are encoded natively (numbers as numbers,
// NULL as null, bool as bool).
func normalizeSQLValue(value any) any {
	if b, ok := value.([]byte); ok {
		return string(b)
	}
	return value
}

func runExternal(cmd *cobra.Command, bin string, args ...string) error {
	process := exec.CommandContext(cmd.Context(), bin, args...)
	process.Stdin = cmd.InOrStdin()
	process.Stdout = cmd.OutOrStdout()
	process.Stderr = cmd.ErrOrStderr()
	return process.Run()
}
