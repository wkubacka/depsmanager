#!/bin/bash

service=$1
api_file=$2

RED='\033[0;31m'
NC='\033[0m' # No Color

usage() {
  echo ""
  echo "USAGE:"
  echo "generate_swagger.sh service_name api_file [dependencies] [options]"
  echo ""
  echo "- service_name: directory name for the service in which generate is invoked"
  echo "- api_file: name of the api file in which api definition doc is present"
  echo "- dependencies: optional, comma separated list of file dependencies to include (e.g. with definitions)"
  echo ""
  echo "OPTIONS"
  echo "--parseDependency, -p   Pass parseDependency to swag generation command"
  echo ""
  echo "For more info see https://github.com/swaggo/swag"
  echo ""
  echo "Preferably use this script in go:generate (e.g. //go:generate ../../generate_swagger.sh authproxy api.go)"
}

if ! command -v swag &>/dev/null; then
  echo -e "${RED}swag not found. Run: "
  echo -e "pushd /tmp; go get github.com/swaggo/swag/cmd/swag; popd${NC}"
  exit 1
fi

if [[ ! $service ]]; then
  echo -e "${RED}required parameter not supplied: service_name${NC}"
  usage
  exit 1
fi

if [[ ! $api_file ]]; then
  echo -e "${RED}required parameter not supplied: api_file${NC}"
  usage
  exit 1
fi

shift;
shift;


while [[ $# -gt 0 ]]; do
  key="$1"

  case $key in
    -p|--parseDependency)
      parse_deps=true
      shift
      shift
      ;;
    *)    # should be deps
      dependencies=$1
      shift
      ;;
  esac
done

git_top_level=$(git rev-parse --show-toplevel)
service_dir=$git_top_level/
docs_dir=$service_dir/docs

swag_dirs=./,$service_dir

if [[ $dependencies ]]; then
  swag_dirs="$swag_dirs,$dependencies"
fi

if $parse_deps; then
  GOFLAGS=-mod=mod swag init -g $api_file -d $swag_dirs -o $docs_dir --parseDependency --parseInternal --parseDepth 10
else
  GOFLAGS=-mod=mod  swag init -g $api_file -d $swag_dirs -o $docs_dir
fi


if [[ $? -ne 0 ]]; then
  echo -e "${RED}Failed to generate api docs for $service.${NC}"
  exit 1
fi

# Only leave .yaml doc
rm $docs_dir/docs.go
rm $docs_dir/swagger.json