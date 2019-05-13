#! /bin/bash

set -euxo pipefail

echo "---- Run tests"
EXIT_CODE=0
docker build -t oasislabs/developer-gateway-ci -f .buildkite/Dockerfile.ci .
docker run \
  --rm \
  --env BUILDKITE_BUILD_NUMBER="$BUILDKITE_BUILD_NUMBER" \
  --env BUILDKITE_PULL_REQUEST="$BUILDKITE_PULL_REQUEST" \
  --env BUILDKITE_BRANCH="$BUILDKITE_BRANCH" \
	--volume="$(pwd)":/app \
	oasislabs/developer-gateway-ci:latest \
	/app/.buildkite/scripts/run_tests.sh || EXIT_CODE=$? ;

# report coverage
echo "--- Uploading Coverage"
set +e
bash <(curl -s https://codecov.io/bash) -Z

if [ $EXIT_CODE -ne 0 ]; then
	exit 1
fi

exit 0
