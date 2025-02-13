# rr
# For AWS STS tokens, ensure you refresh before running tests:
# aws sts assume-role --role-arn <your-role> --role-session-name test-session > credentials.json

# export AWS_ACCESS_KEY_ID=$(jq -r .Credentials.AccessKeyId credentials.json)
# export AWS_SECRET_ACCESS_KEY=$(jq -r .Credentials.SecretAccessKey credentials.json) 
# export AWS_SESSION_TOKEN=$(jq -r .Credentials.SessionToken credentials.json)
