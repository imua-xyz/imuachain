#!/usr/bin/env sh
# shellcheck disable=SC3040
(set -o pipefail 2>/dev/null) && set -o pipefail
set -x

BINARY=/imua/${BINARY:-imuad}
ID=${ID:-0}
LOG=${LOG:-imua.log}

if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found. Please add the binary to the shared folder. Please use the BINARY environment variable if the name of the binary is not 'imuad'"
	exit 1
fi

export EHOME="/home/imua/data/node${ID}/imuad"
export APP_TOML="$EHOME/config/app.toml"
export CLIENT_TOML="$EHOME/config/client.toml"
#APP_TOML_TMP="$EHOME/config/tmp_app.toml"
#CLIENT_TOML_TMP="$EHOME/config/tmp_client.toml"
#cat $APP_TOML | tomlq '.api["enable"]=true' --toml-output > $APP_TOML_TMP && mv $APP_TOML_TMP $APP_TOML
#sed -i.bak 's/chain-id =.*/chain-id = "evmos_9000-8808"/g'  "${CLIENT_TOML_TMP}"

if [ -d "$(dirname "${EHOME}"/"${LOG}")" ]; then
	"${BINARY}" --home "${EHOME}" "$@" | tee "${EHOME}/${LOG}"
else
	"${BINARY}" --home "${EHOME}" "$@"
fi
