// DBDeployer - The MySQL Sandbox
// Copyright © 2006-2018 Giuseppe Maxia
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"

	"github.com/datacharmer/dbdeployer/common"
	"github.com/datacharmer/dbdeployer/concurrent"
	"github.com/datacharmer/dbdeployer/defaults"
	"github.com/datacharmer/dbdeployer/globals"
	"github.com/datacharmer/dbdeployer/sandbox"
	"github.com/spf13/cobra"
)

func deleteSandbox(cmd *cobra.Command, args []string) {
	var execLists []concurrent.ExecutionList
	if len(args) < 1 {
		common.Exit(1,
			"Sandbox name (or \"ALL\") required.",
			"You can run 'dbdeployer sandboxes for a list of available deployments'")
	}
	flags := cmd.Flags()
	sandboxName := args[0]
	confirm, _ := flags.GetBool(globals.ConfirmLabel)
	runConcurrently, _ := flags.GetBool(globals.ConcurrentLabel)
	if common.IsEnvSet("RUN_CONCURRENTLY") {
		runConcurrently = true
	}
	skipConfirm, _ := flags.GetBool(globals.SkipConfirmLabel)
	sandboxDir, err := getAbsolutePathFromFlag(cmd, "sandbox-home")
	common.ErrCheckExitf(err, 1, "error finding absolute path for 'sandbox-home'")

	deletionList := []common.SandboxInfo{{SandboxName: sandboxName, Locked: false}}
	if sandboxName == "ALL" || sandboxName == "all" {
		confirm = true
		if skipConfirm {
			confirm = false
		}
		deletionList, err = common.GetInstalledSandboxes(sandboxDir)
		common.ErrCheckExitf(err, 1, globals.ErrRetrievingSandboxList, err)
	}
	if len(deletionList) == 0 {
		common.CondPrintf("Nothing to delete in %s\n", sandboxDir)
		return
	}
	if len(deletionList) > 60 && runConcurrently {
		fmt.Println("# Concurrency disabled. Can't run more than 60 concurrent operations")
		runConcurrently = false
	}
	common.CondPrintf("List of deployed sandboxes:\n")
	unlockedFound := false
	for _, sb := range deletionList {
		locked := ""
		if sb.Locked {
			locked = "(*LOCKED*)"
		} else {
			unlockedFound = true
		}
		common.CondPrintf("%s/%s %s\n", sandboxDir, sb.SandboxName, locked)
	}
	if !unlockedFound {
		common.CondPrintf("No unlocked sandboxes found.\n")
		return
	}
	if confirm {
		common.CondPrintf("Do you confirm? y/[N] ")

		bio := bufio.NewReader(os.Stdin)
		line, _, err := bio.ReadLine()
		if err != nil {
			fmt.Println(err)
		} else {
			answer := string(line)
			if answer == "y" || answer == "Y" {
				fmt.Println("Proceeding with deletion")
			} else {
				common.Exit(0, "Execution interrupted by user")
			}
		}
	}
	for _, sb := range deletionList {
		if sb.Locked {
			common.CondPrintf("Sandbox %s is locked\n", sb.SandboxName)
		} else {
			execList, err := sandbox.RemoveSandbox(sandboxDir, sb.SandboxName, runConcurrently)
			if err != nil {
				common.Exitf(1, globals.ErrWhileDeletingSandbox, err)
			}
			for _, list := range execList {
				execLists = append(execLists, list)
			}
		}
	}
	concurrent.RunParallelTasksByPriority(execLists)
	for _, sb := range deletionList {
		fullPath := path.Join(sandboxDir, sb.SandboxName)
		if !sb.Locked {
			err := defaults.DeleteFromCatalog(fullPath)
			if err != nil {
				common.Exitf(1, globals.ErrRemovingFromCatalog, fullPath)
			}
		}
	}
}

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete sandbox_name (or \"ALL\")",
	Short:   "delete an installed sandbox",
	Aliases: []string{"remove", "destroy"},
	Example: `
	$ dbdeployer delete msb_8_0_4
	$ dbdeployer delete rsandbox_5_7_21`,
	Long: `Stops the sandbox (and its depending sandboxes, if any), and removes it.
Warning: this command is irreversible!`,
	Run: deleteSandbox,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolP(globals.SkipConfirmLabel, "", false, "Skips confirmation with multiple deletions.")
	deleteCmd.Flags().BoolP(globals.ConfirmLabel, "", false, "Requires confirmation.")
	deleteCmd.Flags().BoolP(globals.ConcurrentLabel, "", false, "Runs multiple deletion tasks concurrently.")
}
