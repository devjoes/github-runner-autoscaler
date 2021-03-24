#!/usr/bin/dumb-init /bin/bash

export RUNNER_ALLOW_RUNASROOT=1
export PATH=$PATH:/actions-runner

_RUNNER_NAME=${RUNNER_NAME:-${RUNNER_NAME_PREFIX:-github-runner}-$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 13 ; echo '')}
_RUNNER_WORKDIR=${RUNNER_WORKDIR:-/_work}
_LABELS=${LABELS:-default}
_RUNNER_GROUP=${RUNNER_GROUP:-Default}
_SHORT_URL=${REPO_URL}

if [[ -n "${ACCESS_TOKEN}" ]]; then
  _TOKEN=$(bash /token.sh)
  RUNNER_TOKEN=$(echo "${_TOKEN}" | jq -r .token)
  _SHORT_URL=$(echo "${_TOKEN}" | jq -r .short_url)
fi

if [[ -n "$RETURN_CONFIG" ]]; then
	echo "Configuring"
	./config.sh \
		--url "${_SHORT_URL}" \
		--token "${RUNNER_TOKEN}" \
		--name "${_RUNNER_NAME}" \
		--work "${_RUNNER_WORKDIR}" \
		--labels "${_LABELS}" \
		--runnergroup "${_RUNNER_GROUP}" \
		--unattended \
		--replace
	set -x
	cp /actions-runner/.runner /actions-runner/.credentials*  /config_output -v
	/RsaParamsToPem/RsaParamsToPem /actions-runner/.credentials_rsaparams private | tee /config_output/private.pem
	/RsaParamsToPem/RsaParamsToPem /actions-runner/.credentials_rsaparams public | tee /config_output/public.pem
	if [[ -n "$OVERWRITE_OUTPUT_UID" ]]; then
		chown "$OVERWRITE_OUTPUT_UID" /config_output -R
	fi
	exit 0
fi

if [[ -d "/actions-creds/$RUNNER_NAME/" ]]; then
	cp /actions-creds/$RUNNER_NAME/.* /actions-runner/ -a
fi

./bin/runsvc.sh