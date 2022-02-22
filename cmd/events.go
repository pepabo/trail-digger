/*
Copyright © GMO Pepabo, inc.

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
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/goccy/go-json"
	"github.com/pepabo/trail-digger/trail"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "show AWS CloudTrail events in order of timeline using trail logs",
	Long:  `show AWS CloudTrail events in order of timeline using trail logs.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn := args[0]
		sess, err := session.NewSession()
		if err != nil {
			return err
		}
		if err := trail.WalkEvents(sess, dsn, opt, func(r *trail.Record) error {
			b, err := json.Marshal(r)
			if err != nil {
				return err
			}
			cmd.Println(string(b))
			return nil
		}); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.Flags().StringVarP(&opt.DatePath, "date", "d", time.Now().Format("2006/01/02"), "target date (eg. 2006/01/02, 2006/01, 2006)")
	eventsCmd.Flags().StringVarP(&opt.StartDatePath, "start-date", "s", "", "start date (eg. 2006/01/02)")
	eventsCmd.Flags().StringVarP(&opt.EndDatePath, "end-date", "e", "", "end date (eg. 2006/01/02)")
	eventsCmd.Flags().StringSliceVarP(&opt.Accounts, "account", "a", []string{}, "target account ID")
	eventsCmd.Flags().StringSliceVarP(&opt.Regions, "region", "r", []string{}, "target region")
	eventsCmd.Flags().BoolVarP(&opt.AllAccounts, "all-accounts", "A", false, "all accounts")
	eventsCmd.Flags().BoolVarP(&opt.AllRegions, "all-regions", "R", false, "all regions")
}
