/*
Copyright © 2019 Canadian Digital Service <max.neuvians@cds-snc.ca>

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
package aws

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/kyokomi/emoji"
	. "github.com/logrusorgru/aurora"
)

func UserPolicyHasAdmin(user *iam.UserDetail, admin string) bool {
	for _, policy := range user.UserPolicyList {
		if *policy.PolicyName == admin {
			return true
		}
	}

	return false
}

func AttachedUserPolicyHasAdmin(user *iam.UserDetail, admin string) bool {
	for _, policy := range user.AttachedManagedPolicies {
		if *policy.PolicyName == admin {
			return true
		}
	}

	return false
}

func GroupPolicyHasAdmin(svc *iam.IAM, group *iam.Group, admin string) bool {
	input := &iam.ListGroupPoliciesInput{
		GroupName: group.GroupName,
	}

	result, err := svc.ListGroupPolicies(input)
	if err != nil {
		fmt.Println("Got error calling ListGroupPolicies for group", group.GroupName)
	}

	// Wade through policies
	for _, policyName := range result.PolicyNames {
		if *policyName == admin {
			return true
		}
	}

	return false
}

func AttachedGroupPolicyHasAdmin(svc *iam.IAM, group *iam.Group, admin string) bool {
	input := &iam.ListAttachedGroupPoliciesInput{GroupName: group.GroupName}
	result, err := svc.ListAttachedGroupPolicies(input)
	if err != nil {
		fmt.Println("Got error getting attached group policies:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	for _, policy := range result.AttachedPolicies {
		if *policy.PolicyName == admin {
			return true
		}
	}

	return false
}

func UsersGroupsHaveAdmin(svc *iam.IAM, user *iam.UserDetail, admin string) bool {
	input := &iam.ListGroupsForUserInput{UserName: user.UserName}
	result, err := svc.ListGroupsForUser(input)
	if err != nil {
		fmt.Println("Got error getting groups for user:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	for _, group := range result.Groups {
		groupPolicyHasAdmin := GroupPolicyHasAdmin(svc, group, admin)

		if groupPolicyHasAdmin {
			return true
		}

		attachedGroupPolicyHasAdmin := AttachedGroupPolicyHasAdmin(svc, group, admin)

		if attachedGroupPolicyHasAdmin {
			return true
		}
	}

	return false
}

func IsUserAdmin(svc *iam.IAM, user *iam.UserDetail, admin string) bool {
	// Check policy, attached policy, and groups (policy and attached policy)
	policyHasAdmin := UserPolicyHasAdmin(user, admin)
	if policyHasAdmin {
		return true
	}

	attachedPolicyHasAdmin := AttachedUserPolicyHasAdmin(user, admin)
	if attachedPolicyHasAdmin {
		return true
	}

	userGroupsHaveAdmin := UsersGroupsHaveAdmin(svc, user, admin)
	if userGroupsHaveAdmin {
		return true
	}

	return false
}

func CheckAdminUsers(key string, secret string) bool {

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
	})

	if err != nil {
		fmt.Println("Error", err)
		return false
	}

	fmt.Println(Green("Checking AWS for users with admin policies attached ..."))

	svc := iam.New(sess)

	numUsers := 0
	numAdmins := 0

	// Get list of users
	user := "User"
	input := &iam.GetAccountAuthorizationDetailsInput{Filter: []*string{&user}}
	resp, err := svc.GetAccountAuthorizationDetails(input)
	if err != nil {
		fmt.Println("Got error getting account details")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// The policy name that indicates administrator access
	adminName := "AdministratorAccess"

	// Wade through resulting users
	for _, user := range resp.UserDetailList {
		numUsers += 1

		isAdmin := IsUserAdmin(svc, user, adminName)

		if isAdmin {
			// fmt.Println(*user.UserName)
			numAdmins += 1
		}
	}

	// Are there more?
	for *resp.IsTruncated {
		input := &iam.GetAccountAuthorizationDetailsInput{Filter: []*string{&user}, Marker: resp.Marker}
		resp, err = svc.GetAccountAuthorizationDetails(input)
		if err != nil {
			fmt.Println("Got error getting account details")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		// Wade through resulting users
		for _, user := range resp.UserDetailList {
			numUsers += 1

			isAdmin := IsUserAdmin(svc, user, adminName)

			if isAdmin {
				fmt.Println(*user.UserName)
				numAdmins += 1
			}
		}
	}

	if numAdmins > 0 {
		emoji.Println(" :exclamation: ", Sprintf(BrightYellow("%d user(s) have admin policies attached"), numAdmins))
	} else {
		emoji.Println(" :white_check_mark: ", BrightGreen("No user accounts have admin policies attached"))
	}

	return true

}
