package cmd

import (
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
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
	queryFile  string
	outputFile string
	writeMode  bool
}

var neo4jCmd = &cobra.Command{
	Use:   "neo4j",
	Short: "Execute file-based Cypher queries against a Neo4j database",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		password := cmp.Or(neo4jCmdFlags.password, os.Getenv("NITS_NEO4J_PASSWORD"), "p4SSw0rd")

		u.PrintRunning("Executing Neo4j queries...")
		results, err := interactions.ExecuteNeo4jQueriesFromFile(ctx, neo4jCmdFlags.uri, neo4jCmdFlags.user, password, neo4jCmdFlags.database, neo4jCmdFlags.queryFile, neo4jCmdFlags.writeMode)
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal("failed to execute neo4j queries", err)
		}

		jsonData, err := json.Marshal(results, jsontext.WithIndent("  "))
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
	neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.uri, "uri", "r", cmp.Or(os.Getenv("NITS_NEO4J_URI"), "neo4j://localhost:7687"), "Neo4j URI (or NITS_NEO4J_URI env)")
	neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.user, "user", "u", cmp.Or(os.Getenv("NITS_NEO4J_USER"), "neo4j"), "Neo4j user (or NITS_NEO4J_USER env)")
	neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.password, "password", "p", "", "Neo4j password (or NITS_NEO4J_PASSWORD env, default p4SSw0rd)")
	neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.database, "database", "d", "neo4j", "Neo4j database")
	neo4jCmd.Flags().StringVar(&neo4jCmdFlags.queryFile, "query-file", "", "Path to a YAML file with a list of Cypher queries")
	neo4jCmd.Flags().StringVarP(&neo4jCmdFlags.outputFile, "output-file", "o", "neo4j-query-result.json", "Output file for the query results")
	neo4jCmd.Flags().BoolVar(&neo4jCmdFlags.writeMode, "write", false, "Open connection in write mode")
	neo4jCmd.MarkFlagRequired("query-file")
}
