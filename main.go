package main

import (
	"net/url"
	"strconv"
	"strings"

	otx "github.com/appscode/osm/context"
	"github.com/appscode/go/types"
	api "github.com/appscode/kubed/pkg/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/graymeta/stow/s3"
	"github.com/graymeta/stow"
	"log"
	"encoding/json"
	"fmt"
	"os"
)

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omiempty"`
	Prefix   string `json:"prefix,omitempty"`
}

func main() {
	config := map[string][]byte{
		api.AWS_ACCESS_KEY_ID: []byte(os.Getenv(api.AWS_ACCESS_KEY_ID)),
		api.AWS_SECRET_ACCESS_KEY: []byte(os.Getenv(api.AWS_SECRET_ACCESS_KEY)),
	}
	nc := &otx.Context{
		Name:   "kubedb",
		Config: stow.ConfigMap{},
	}
	spec := S3Spec{
		Endpoint: "s3.amazonaws.com",
		Bucket: "kubed22",
	}

	keyID, foundKeyID := config[api.AWS_ACCESS_KEY_ID]
	key, foundKey := config[api.AWS_SECRET_ACCESS_KEY]
	if foundKey && foundKeyID {
		nc.Config[s3.ConfigAccessKeyID] = string(keyID)
		nc.Config[s3.ConfigSecretKey] = string(key)
		nc.Config[s3.ConfigAuthType] = "accesskey"
	} else {
		nc.Config[s3.ConfigAuthType] = "iam"
	}
	if strings.HasSuffix(spec.Endpoint, ".amazonaws.com") {
		// find region
		var sess *session.Session
		var err error
		if nc.Config[s3.ConfigAuthType] == "iam" {
			sess, err = session.NewSessionWithOptions(session.Options{
				Config: *aws.NewConfig(),
				// Support MFA when authing using assumed roles.
				SharedConfigState:       session.SharedConfigEnable,
				AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
			})
		} else {
			config := &aws.Config{
				Credentials: credentials.NewStaticCredentials(string(keyID), string(key), ""),
				Region:      aws.String("us-east-1"),
			}
			sess, err = session.NewSessionWithOptions(session.Options{
				Config: *config,
				// Support MFA when authing using assumed roles.
				SharedConfigState:       session.SharedConfigEnable,
				AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
			})
		}
		if err != nil {
			log.Fatalln(err)
		}
		svc := _s3.New(sess)
		out, err := svc.GetBucketLocation(&_s3.GetBucketLocationInput{
			Bucket: types.StringP(spec.Bucket),
		})
		nc.Config[s3.ConfigRegion] = types.String(out.LocationConstraint)
	} else {
		nc.Config[s3.ConfigEndpoint] = spec.Endpoint
		if u, err := url.Parse(spec.Endpoint); err == nil {
			nc.Config[s3.ConfigDisableSSL] = strconv.FormatBool(u.Scheme == "http")
		}
	}

	b, _ := json.MarshalIndent(nc, "", "  ")
	fmt.Println(string(b))
}
