/*
Copyright © 2022 GMO Pepabo, inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/olekukonko/tablewriter"
	"github.com/pepabo/trail-digger/trail"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "analyze AWS CloudTrail events using trail logs",
	Long:  `analyze AWS CloudTrail events using trail logs.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn := args[0]
		sess, err := session.NewSession()
		if err != nil {
			return err
		}
		eventTypeCount := map[string]int{
			"ManagementEvent": 0,
			"DataEvent":       0,
		}
		eventSourceCount := map[string]int{}
		regionCount := map[string]int{}
		recipientAccountIDCount := map[string]int{}

		var mu sync.Mutex
		if err := trail.WalkEvents(sess, dsn, opt, func(r *trail.Record) error {
			mu.Lock()
			if r.ManagementEvent {
				eventTypeCount["ManagementEvent"] += 1
			} else {
				eventTypeCount["DataEvent"] += 1
			}
			eventSourceCount[r.EventSource] += 1
			regionCount[r.AwsRegion] += 1
			recipientAccountIDCount[r.RecipientAccountID] += 1
			mu.Unlock()
			return nil
		}); err != nil {
			return err
		}

		data := [][]string{}
		data = append(data, []string{"", "", ""})

		{
			// Event Type
			data = append(data, []string{"Event Type", "Management Event:", strconv.Itoa(eventTypeCount["ManagementEvent"])})
			data = append(data, []string{"Event Type", "Data Event:", strconv.Itoa(eventTypeCount["DataEvent"])})
			data = append(data, []string{"", "", ""})
		}

		{
			// Event Source
			keys := []string{}
			for key := range eventSourceCount {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				data = append(data, []string{"Event Source", fmt.Sprintf("%s:", key), strconv.Itoa(eventSourceCount[key])})
			}
			data = append(data, []string{"", "", ""})
		}

		{
			// Region
			keys := []string{}
			for key := range regionCount {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				data = append(data, []string{"Region", fmt.Sprintf("%s:", key), strconv.Itoa(regionCount[key])})
			}
			data = append(data, []string{"", "", ""})
		}

		{
			// Recipient Account ID
			keys := []string{}
			for key := range recipientAccountIDCount {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				data = append(data, []string{"Recipient Account ID", fmt.Sprintf("%s:", key), strconv.Itoa(recipientAccountIDCount[key])})
			}
			data = append(data, []string{"", "", ""})
		}

		cmd.Println("")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"", "", "Count"})
		table.SetAutoWrapText(false)
		table.SetAutoFormatHeaders(false)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")
		table.SetHeaderLine(false)
		table.SetBorder(false)
		table.SetAutoMergeCellsByColumnIndex([]int{0})
		table.AppendBulk(data)
		table.Render()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVarP(&opt.DatePath, "date", "d", time.Now().Format("2006/01/02"), "target date (eg. 2006/01/02, 2006/01, 2006)")
	analyzeCmd.Flags().StringVarP(&opt.StartDatePath, "start-date", "s", "", "start date (eg. 2006/01/02)")
	analyzeCmd.Flags().StringVarP(&opt.EndDatePath, "end-date", "e", "", "end date (eg. 2006/01/02)")
	analyzeCmd.Flags().StringSliceVarP(&opt.Accounts, "account", "a", []string{}, "target account ID")
	analyzeCmd.Flags().StringSliceVarP(&opt.Regions, "region", "r", []string{}, "target region")
	analyzeCmd.Flags().BoolVarP(&opt.AllAccounts, "all-accounts", "A", false, "all accounts")
	analyzeCmd.Flags().BoolVarP(&opt.AllRegions, "all-regions", "R", false, "all regions")
	analyzeCmd.Flags().StringVarP(&opt.LogFilePrefix, "log-file-prefix", "p", "", "log file prefix")
}
