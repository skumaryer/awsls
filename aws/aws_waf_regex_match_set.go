// Code is generated. DO NOT EDIT.

package aws

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/waf"
)

func ListWafRegexMatchSet(client *Client) {
	req := client.wafconn.ListRegexMatchSetsRequest(&waf.ListRegexMatchSetsInput{})

	resp, err := req.Send(context.Background())
	if err != nil {
		log.Printf("aws_waf_regex_match_set: %s", err)
	} else {
		if len(resp.RegexMatchSets) > 0 {
			fmt.Println("")
			fmt.Println("aws_waf_regex_match_set:")
			for _, r := range resp.RegexMatchSets {
				fmt.Println(*r.RegexMatchSetId)

			}
		}
	}

}
