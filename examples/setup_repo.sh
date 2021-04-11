#!/bin/bash
set -x
build_count=10
repo_count=1
admin_user=devjoes
admin_token=`cat ~/devjoes_repo_all_token`
read_token=`cat ~/devjoes_repo_all_token`
examples=`realpath .`
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
		cp $examples/simple-app/Dockerfile . -v
		cp $examples/simple-app/.github ./ -av
		ls -a
		git add .
		git commit -m "add app"
		git push --set-upstream origin master
		all_repos+=($repo)
		cd -
	done

	for r in ${all_repos[@]}; do
		cd $r
		cp $examples/resources/* ./ -a
		sed -i "s/placeholdername/$r/g" *.yaml
		node $examples/add-runner.js -n "build-$r" -o "$admin_user" -r "$r" -m 4 -p "$read_token" -a "$admin_token" -f build.yaml
		node $examples/add-runner.js -n "deploy-$r" -o "$admin_user" -r "$r" -m 4 -p "$read_token" -a "$admin_token" -l "deploy,test" -f deploy.yaml
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

	read -p "Press enter to delete test repos"
	for r in ${all_repos[@]}; do
		kubectl delete ns $r
		GITHUB_TOKEN="$admin_token" hub delete "$admin_user/$r"
	done
)