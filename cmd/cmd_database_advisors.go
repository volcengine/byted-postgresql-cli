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
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/cobra"
)

type advisorFinding struct {
	Severity    string `json:"severity" yaml:"severity"`
	Category    string `json:"category" yaml:"category"`
	Object      string `json:"object,omitempty" yaml:"object,omitempty"`
	Message     string `json:"message" yaml:"message"`
	Remediation string `json:"remediation,omitempty" yaml:"remediation,omitempty"`
}

type advisorReport struct {
	Summary  advisorSummary   `json:"summary" yaml:"summary"`
	Findings []advisorFinding `json:"findings" yaml:"findings"`
}

type advisorSummary struct {
	Critical int `json:"critical" yaml:"critical"`
	Warning  int `json:"warning" yaml:"warning"`
	Info     int `json:"info" yaml:"info"`
	Skipped  int `json:"skipped" yaml:"skipped"`
}

type advisorCheck struct {
	category string
	query    string
	optional bool
}

var advisorChecks = []advisorCheck{
	{
		category: "dead-tuples",
		query: `SELECT 'warning', 'dead-tuples', schemaname || '.' || relname,
	format('Table has %s dead tuples.', n_dead_tup),
		'Run VACUUM (ANALYZE) on the table.'
FROM pg_stat_user_tables
WHERE n_dead_tup > 0 AND n_dead_tup > GREATEST(n_live_tup, 1000) * 0.2
ORDER BY n_dead_tup DESC;`,
	},
	{
		category: "missing-primary-key",
		query: `SELECT 'warning', 'missing-primary-key', n.nspname || '.' || c.relname,
	'Table has no primary key.',
	'Add a primary key if the table is used for application data or replication.'
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r' AND n.nspname NOT IN ('pg_catalog', 'information_schema')
	AND NOT EXISTS (
		SELECT 1 FROM pg_constraint con
		WHERE con.conrelid = c.oid AND con.contype = 'p'
	)
ORDER BY 3;`,
	},
	{
		category: "unused-index",
		query: `SELECT 'info', 'unused-index', schemaname || '.' || indexrelname,
	format('Index has not been scanned since statistics reset (idx_scan=%s).', idx_scan),
	'Review the index before removing it; zero scans do not prove that it is safe to drop.'
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY 3;`,
	},
	{
		category: "long-transaction",
		query: `SELECT 'warning', 'long-transaction', pid::text,
	format('Transaction has been open for %s.', now() - xact_start),
	'Commit or roll back the transaction; investigate sessions that remain open.'
FROM pg_stat_activity
WHERE xact_start IS NOT NULL AND now() - xact_start > interval '5 minutes'
ORDER BY xact_start;`,
	},
	{
		category: "slow-query",
		optional: true,
		query: `SELECT 'warning', 'slow-query', left(query, 120),
	format('Query mean execution time is %s ms across %s calls.', round(mean_exec_time::numeric, 2), calls),
	'Inspect the query plan and consider indexes or query changes.'
FROM pg_stat_statements
WHERE calls > 0
ORDER BY mean_exec_time DESC
LIMIT 20;`,
	},
}

func runDatabaseAdvisors(cmd *cobra.Command, target databaseTarget, format string) error {
	connectionURL, err := resolveDatabaseTarget(cmd, target)
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", connectionURL)
	if err != nil {
		return fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(cmd.Context()); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	report := advisorReport{}
	for _, check := range advisorChecks {
		findings, checkErr := runAdvisorCheck(cmd, db, check)
		if checkErr != nil {
			if check.optional && isMissingPgStatStatements(checkErr) {
				report.Findings = append(report.Findings, advisorFinding{
					Severity: "skipped", Category: check.category,
					Message:     "pg_stat_statements is not enabled; slow-query check was skipped.",
					Remediation: "Ask a DBA to enable pg_stat_statements, then rerun db advisors.",
				})
				continue
			}
			report.Findings = append(report.Findings, advisorFinding{
				Severity: "skipped", Category: check.category,
				Message: "Check was skipped: " + checkErr.Error(),
			})
			continue
		}
		report.Findings = append(report.Findings, findings...)
	}
	for _, finding := range report.Findings {
		switch finding.Severity {
		case "critical":
			report.Summary.Critical++
		case "warning":
			report.Summary.Warning++
		case "info":
			report.Summary.Info++
		case "skipped":
			report.Summary.Skipped++
		}
	}
	return writeAdvisorReport(cmd, report, format)
}

func runAdvisorCheck(cmd *cobra.Command, db *sql.DB, check advisorCheck) ([]advisorFinding, error) {
	rows, err := db.QueryContext(cmd.Context(), check.query)
	if err != nil {
		return nil, translatePostgreSQLQueryError(err)
	}
	defer rows.Close()
	var findings []advisorFinding
	for rows.Next() {
		var finding advisorFinding
		if err := rows.Scan(&finding.Severity, &finding.Category, &finding.Object, &finding.Message, &finding.Remediation); err != nil {
			return nil, err
		}
		findings = append(findings, finding)
	}
	return findings, rows.Err()
}

func isMissingPgStatStatements(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01" &&
		strings.Contains(pgErr.Message, `relation "pg_stat_statements" does not exist`)
}

func writeAdvisorReport(cmd *cobra.Command, report advisorReport, format string) error {
	writer := fromCtx(cmd).Writer()
	if format == "unaligned" {
		for _, finding := range report.Findings {
			fmt.Fprintf(cmd.OutOrStdout(), "%s|%s|%s|%s|%s\n",
				finding.Severity, finding.Category, finding.Object, finding.Message, finding.Remediation)
		}
		return nil
	}
	if format == "json" || format == "yaml" {
		return writer.WriteItem(report, nil)
	}
	fields := []string{"Severity", "Category", "Object", "Message", "Remediation"}
	return writer.WriteList(report.Findings, fields)
}
