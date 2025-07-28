#!/bin/bash

set -e

if [[ -z "$1" ]]; then
  echo "usage: make_release.sh version|major|minor|patch|auto"
  exit
fi

echo "Current changelog:"

changie batch $1 -d
echo

git_tag=$(git describe --tags --abbrev=0)
rev=$(git rev-list --tags --max-count=1)
tag=$(changie latest)

#if [[ "${git_tag}" != "${tag}" ]]; then
#	echo "last git tag is ${git_tag}, but changie reports ${tag}"
#	exit 1
#fi

if [[ -n "${rev}" ]]; then
        prev_tag=$(git describe --tags ${rev})
        echo "commits from ${prev_tag}:"
        echo "===================================================="
        git log --color --graph --pretty=format:'%Cred%h%Creset -%C(yellow)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset' --abbrev-commit ${rev}..HEAD
else
        echo "no tags"
fi

while true; do
    read -p "Continue? " yn
    case $yn in
        [Yy]* ) break;;
        [Nn]* ) exit;;
        * ) echo "Please answer yes or no.";;
    esac
done

changie batch $1
changie merge
git add .changes/*
git add CHANGELOG.md

git commit -am 'changelog'
git tag "$(changie latest)"

while true; do
    read -p "Push? " yn
    case $yn in
        [Yy]* ) break;;
        [Nn]* ) exit;;
        * ) echo "Please answer yes or no.";;
    esac
done

git push --tags
