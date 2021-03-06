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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/kyokomi/emoji"
	. "github.com/logrusorgru/aurora"
)

func CheckUserMFA(key string, secret string) bool {

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
	})

	if err != nil {
		fmt.Println("Error", err)
		return false
	}

	fmt.Println(Green("Checking AWS users accounts for MFA ..."))

	svc := iam.New(sess)

	result, err := svc.ListUsers(&iam.ListUsersInput{})

	if err != nil {
		fmt.Println("Error", err)
		return false
	}

	mfaAccounts := 0

	for _, user := range result.Users {
		if user == nil {
			continue
		}
		devices, err := svc.ListMFADevices(&iam.ListMFADevicesInput{
			UserName: user.UserName,
		})
		if err != nil {
			fmt.Println("Error", err)
			return false
		}
		if len(devices.MFADevices) != 0 {
			mfaAccounts++
		}
	}

	if len(result.Users) != mfaAccounts {
		emoji.Println(" :exclamation: ", Sprintf(BrightYellow("%d out of %d users have MFA active"), mfaAccounts, len(result.Users)))
	} else {
		emoji.Println(" :white_check_mark: ", BrightGreen("All user accounts use MFA"))
	}

	return true
}
