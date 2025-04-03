module github.com/sfomuseum/go-sfomuseum-geo

// replace github.com/tidwall/sjson v1.2.5 => github.com/sfomuseum/sjson v0.0.0-20250403211343-34be55140410

go 1.24

toolchain go1.24.0

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/paulmach/orb v0.11.1
	github.com/sfomuseum/go-flags v0.10.0
	github.com/sfomuseum/go-geojson-geotag/v2 v2.0.0
	github.com/sfomuseum/go-sfomuseum-writer/v3 v3.0.3
	github.com/tidwall/gjson v1.18.0
	github.com/tidwall/sjson v1.2.5
	github.com/whosonfirst/go-ioutil v1.0.2
	github.com/whosonfirst/go-reader v1.0.2
	github.com/whosonfirst/go-reader-findingaid v0.14.2
	github.com/whosonfirst/go-reader-github v0.6.11
	github.com/whosonfirst/go-whosonfirst-export/v2 v2.8.4
	github.com/whosonfirst/go-whosonfirst-feature v0.0.28
	github.com/whosonfirst/go-whosonfirst-format v0.4.1
	github.com/whosonfirst/go-whosonfirst-id v1.3.0
	github.com/whosonfirst/go-whosonfirst-reader v1.0.2
	github.com/whosonfirst/go-whosonfirst-uri v1.3.0
	github.com/whosonfirst/go-writer-featurecollection/v3 v3.0.2
	github.com/whosonfirst/go-writer-github/v3 v3.0.4
	github.com/whosonfirst/go-writer/v3 v3.1.1
	gocloud.dev v0.41.0
)

require (
	github.com/aaronland/go-artisanal-integers v0.9.1 // indirect
	github.com/aaronland/go-aws-auth v1.3.1 // indirect
	github.com/aaronland/go-aws-dynamodb v0.0.5 // indirect
	github.com/aaronland/go-aws-session v0.2.1 // indirect
	github.com/aaronland/go-brooklynintegers-api v1.2.7 // indirect
	github.com/aaronland/go-pool/v2 v2.0.0 // indirect
	github.com/aaronland/go-roster v1.0.0 // indirect
	github.com/aaronland/go-string v1.0.0 // indirect
	github.com/aaronland/go-uid v0.4.0 // indirect
	github.com/aaronland/go-uid-artisanal v0.0.4 // indirect
	github.com/aaronland/go-uid-proxy v0.3.1 // indirect
	github.com/aaronland/go-uid-whosonfirst v0.0.5 // indirect
	github.com/aws/aws-sdk-go v1.55.6 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.12 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.65 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.21.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/iam v1.28.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.58.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.17 // indirect
	github.com/aws/smithy-go v1.22.3 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/g8rswimmer/error-chain v1.0.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/go-github/v48 v48.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/google/wire v0.6.0 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jtacoma/uritemplates v1.0.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/sfomuseum/go-edtf v1.2.1 // indirect
	github.com/sfomuseum/go-sfomuseum-export/v2 v2.3.11 // indirect
	github.com/sfomuseum/runtimevar v1.2.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/whosonfirst/go-reader-http v0.3.1 // indirect
	github.com/whosonfirst/go-whosonfirst-findingaid/v2 v2.7.1 // indirect
	github.com/whosonfirst/go-whosonfirst-flags v0.5.1 // indirect
	github.com/whosonfirst/go-whosonfirst-sources v0.1.0 // indirect
	github.com/whosonfirst/go-whosonfirst-writer/v3 v3.1.3 // indirect
	go.mongodb.org/mongo-driver v1.11.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/ratelimit v0.3.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/oauth2 v0.28.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/api v0.228.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
