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
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/docker/go-units"
	"github.com/pepabo/trail-digger/trail"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var keyRe = regexp.MustCompile(`/([0-9]+)/CloudTrail/([a-z\-]+\-[123])/`)

var sizeCmd = &cobra.Command{
	Use:   "size",
	Short: "show size of trail logs",
	Long:  `show size of trail logs.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn := args[0]
		sess, err := session.NewSession()
		if err != nil {
			return err
		}
		size := int64(0)
		regionCount := map[string]int64{}
		accountIDCount := map[string]int64{}
		var mu sync.Mutex
		if err := trail.WalkObjects(sess, dsn, opt, func(o *s3.Object) error {
			matches := keyRe.FindAllStringSubmatch(*o.Key, 1)
			mu.Lock()
			size += *o.Size
			regionCount[matches[0][2]] += *o.Size
			accountIDCount[matches[0][1]] += *o.Size
			mu.Unlock()
			return nil
		}); err != nil {
			return err
		}

		data := [][]string{}
		data = append(data, []string{"", "", ""})

		{
			// Region
			keys := []string{}
			for key := range regionCount {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				data = append(data, []string{"Region", fmt.Sprintf("%s:", key), fmt.Sprintf("%s (%dB)", units.BytesSize(float64(regionCount[key])), regionCount[key])})
			}
			data = append(data, []string{"", "", ""})
		}

		{
			// AccountID
			keys := []string{}
			for key := range accountIDCount {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				data = append(data, []string{"Account ID", fmt.Sprintf("%s:", key), fmt.Sprintf("%s (%dB)", units.BytesSize(float64(accountIDCount[key])), accountIDCount[key])})
			}
			data = append(data, []string{"", "", ""})
		}

		data = append(data, []string{"Total", "", fmt.Sprintf("%s (%dB)", units.BytesSize(float64(size)), size)})

		cmd.Println("")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"", "", "Size"})
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
	rootCmd.AddCommand(sizeCmd)
	sizeCmd.Flags().StringVarP(&opt.DatePath, "date", "d", time.Now().Format("2006/01/02"), "target date (eg. 2006/01/02, 2006/01, 2006)")
	sizeCmd.Flags().StringSliceVarP(&opt.Accounts, "account", "a", []string{}, "target account ID")
	sizeCmd.Flags().StringSliceVarP(&opt.Regions, "region", "r", []string{}, "target region")
	sizeCmd.Flags().BoolVarP(&opt.AllAccounts, "all-accounts", "A", false, "all accounts")
	sizeCmd.Flags().BoolVarP(&opt.AllRegions, "all-regions", "R", false, "all regions")
}
