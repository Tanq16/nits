package interactionsCmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/interactions"
	u "github.com/tanq16/nits/utils"
)

var neo4jCmdFlags struct {
	uri        string
	user       string
	password   string
	database   string
	query      string
	queryFile  string
	outputFile string
	writeMode  bool
}

var Neo4jCmd = &cobra.Command{
	Use:   "neo4j",
	Short: "Execute inline or file-based Cypher queries against a Neo4j database",
	Run: func(cmd *cobra.Command, args []string) {
		if neo4jCmdFlags.query != "" && neo4jCmdFlags.queryFile != "" {
			u.PrintFatal("please provide either a query or a query file, not both", nil)
		}
		if neo4jCmdFlags.query == "" && neo4jCmdFlags.queryFile == "" {
			u.PrintFatal("a query or a query file is required", nil)
		}

		ctx := context.Background()
		var results []interactions.QueryResult
		var err error

		u.PrintRunning("Executing Neo4j query/queries...")
		if neo4jCmdFlags.query != "" {
			results, err = interactions.ExecuteNeo4jQueries(ctx, neo4jCmdFlags.uri, neo4jCmdFlags.user, neo4jCmdFlags.password, neo4jCmdFlags.database, []string{neo4jCmdFlags.query}, neo4jCmdFlags.writeMode)
		} else {
			results, err = interactions.ExecuteNeo4jQueriesFromFile(ctx, neo4jCmdFlags.uri, neo4jCmdFlags.user, neo4jCmdFlags.password, neo4jCmdFlags.database, neo4jCmdFlags.queryFile, neo4jCmdFlags.writeMode)
		}
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal("failed to execute neo4j queries", err)
		}

		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			u.PrintFatal("failed to marshal results to JSON", err)
		}
		if err := os.WriteFile(neo4jCmdFlags.outputFile, jsonData, 0644); err != nil {
			u.PrintFatal(fmt.Sprintf("failed to write results to file: %s", neo4jCmdFlags.outputFile), err)
		}
		u.PrintSuccess(fmt.Sprintf("Executed queries and saved results to %s", neo4jCmdFlags.outputFile))
	},
}

func init() {
	Neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.uri, "uri", "r", "neo4j://localhost:7687", "Neo4j URI")
	Neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.user, "user", "u", "neo4j", "Neo4j user")
	Neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.password, "password", "p", "p4SSw0rd", "Neo4j password")
	Neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.database, "database", "d", "neo4j", "Neo4j database")
	Neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.query, "query", "q", "", "Single Cypher query to execute")
	Neo4jCmd.Flags().StringVar(&neo4jCmdFlags.queryFile, "query-file", "", "Path to a YAML file with a list of Cypher queries")
	Neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.outputFile, "output-file", "o", "neo4j-query-result.json", "Output file for the query results")
	Neo4jCmd.Flags().BoolVar(&neo4jCmdFlags.writeMode, "write", false, "Open connection in write mode")
}
