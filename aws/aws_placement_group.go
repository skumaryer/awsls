// Code is generated. DO NOT EDIT.

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func ListPlacementGroup(client *Client) error {
	req := client.ec2conn.DescribePlacementGroupsRequest(&ec2.DescribePlacementGroupsInput{})

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}

	if len(resp.PlacementGroups) > 0 {
		for _, r := range resp.PlacementGroups {
			fmt.Println(*r.GroupName)
			for _, t := range r.Tags {
				fmt.Printf("\t%s: %s\n", *t.Key, *t.Value)
			}
		}
	}

	return nil
}