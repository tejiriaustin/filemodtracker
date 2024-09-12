/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"database/sql"
	"fmt"
	"github.com/spf13/cobra"
)

// tinyCmd represents the tiny command
var tinyCmd = &cobra.Command{
	Use:   "tiny",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tiny called")

		database, err := sql.Open("sqlite3", ".tmp")
		if err != nil {
			fmt.Println(err.Error())
		}

		query := `CREATE TABLE IF NOT EXISTS file_events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        path TEXT,
        operation TEXT,
        timestamp DATETIME
    )`

		_, err = database.Exec(query)
		if err != nil {
			fmt.Println(err.Error())
		}

		return
	},
}

func init() {
	rootCmd.AddCommand(tinyCmd)
}
