#!/bin/bash
#set -x
build_count=10
repo_count=4
admin_user=devjoes
admin_token=`cat ~/devjoes_repo_all_token`
read_token=`cat ~/devjoes_repo_all_token`
folder=`realpath .`
TMP=`mktemp -d`
(
	all_repos=()
	cd "$TMP"
	for i in $(seq 1 $repo_count);do
		repo="test-repo-$i-$RANDOM"
		mkdir -p "$repo"
		cd "$repo"
		git init
		GITHUB_TOKEN="$admin_token" hub create "$admin_user/$repo"
		cp $folder/simple-app/Dockerfile . -v
		cp $folder/simple-app/.github ./ -av
		ls -a
		git add .
		git commit -m "add app"
		# This is just if you have multiple accounts
		sed -i 's/git@github.com:devjoes/git@github.com-devjoes:devjoes/' .git/config
		git push --set-upstream origin master
		all_repos+=($repo)
		cd -
	done

	for r in ${all_repos[@]}; do
		cd $r
		cp $folder/resources/* ./ -a
		sed -i "s/placeholdername/$r/g" *.yaml
		failed=1
		while [[ "$failed" != "0" ]]; do
			node $folder/../add-runner.js -n "build-$r" -o "$admin_user" -r "$r" -m 2 -p "$read_token" -a "$admin_token" -l "build" -f build.yaml
			failed=$?
		done
		failed=1
		while [[ "$failed" != "0" ]]; do
			node $folder/../add-runner.js -n "deploy-$r" -o "$admin_user" -r "$r" -m 1 -p "$read_token" -a "$admin_token" -l "deploy" -f deploy.yaml
			failed=$?
		done

		kubectl apply -k .
		cd -
	done

	read -p "Press enter to queue jobs"
	for r in ${all_repos[@]}; do
		for i in `seq 1 $build_count`; do
		curl --request POST \
		--url "https://api.github.com/repos/$admin_user/$r/dispatches" \
		--header "authorization: Bearer $admin_token" \
		--data "{\"event_type\": \"build_$i\"}"
		done
	done

	started=0
	ended=0
	attempts=0
	while [[ "$started" == "0" || "$ended" == "0" ]] && [[ "$attempts" != "300" ]]; do
		attempts=`expr $attempts + 1`
		count=0
		for r in ${all_repos[@]}; do
			builds=`kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/$r/Scaledactionrunners/build-$r/wf_runs_on_build" | jq '.items[0].value' -r`
			deploys=`kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/$r/Scaledactionrunners/deploy-$r/wf_runs_on_deploy" | jq '.items[0].value' -r`
			if [[ -n "$builds" && -n "$deploys" ]]; then
				printf "$attempts\t$r\t$builds\t$deploys\n"
				count=`expr $count + $builds + $deploys`
			fi
			sleep 1s
		done
		if [[ "$count" != "0" ]]; then
			started=1
		fi
		if [[ "$count" == "0" ]] && [[ "$started" == "1" ]]; then
			ended=1
		fi
	done
	read -p "Press enter to delete test repos"
	for r in ${all_repos[@]}; do
		kubectl delete ns $r
		GITHUB_TOKEN="$admin_token" hub delete "$admin_user/$r"
	done
)