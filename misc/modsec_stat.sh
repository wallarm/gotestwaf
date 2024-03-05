#!/usr/bin/env sh

RESULT_OUTPUT="modsec_stat_$(date -u +"%Y-%m-%dT%H:%M:%SZ").txt"
GTW_WORKERS_NUMBER=10

ENDPOINT="localhost"
if [ "$(uname -s)" = "Darwin" ]; then
	ENDPOINT="host.docker.internal"
fi

docker_wait() {
	CONTAINER_STATUS="$(docker inspect --format "{{ json .State.Health.Status }}" "$1")"
	until [ "$CONTAINER_STATUS" = '"healthy"' ]
	do
		echo "Waiting for container to start..."
		sleep 5
		CONTAINER_STATUS="$(docker inspect --format "{{ json .State.Health.Status }}" "$1")"
	done
}

docker pull mendhak/http-https-echo:31
docker pull owasp/modsecurity-crs:nginx-alpine

docker run --rm -d -p 8088:8080 --name gotestwaf_test_app mendhak/http-https-echo:31

TEST_APP_ADDRESS="$(docker inspect --format '{{ .NetworkSettings.IPAddress }}' gotestwaf_test_app)"

git describe --dirty > "$RESULT_OUTPUT"

for PARANOIA in $(seq 1 4); do
	docker run --rm -d \
		-p 8080:8080 -p 8443:8443 \
		--name "modsec_paranoia_$PARANOIA" \
		-e PARANOIA="$PARANOIA" \
		-e BLOCKING_PARANOIA="$PARANOIA" \
		-e EXECUTING_PARANOIA="$PARANOIA" \
		-e DETECTION_PARANOIA="$PARANOIA" \
		-e BACKEND="http://${TEST_APP_ADDRESS}:8080" \
		owasp/modsecurity-crs:nginx-alpine

	docker_wait "modsec_paranoia_$PARANOIA"

	OUTPUT=$(\
		docker run --rm \
			--network="host" \
			-v "$(pwd)/reports:/app/reports" \
			gotestwaf \
				--url="http://${ENDPOINT}:8080/" \
				--workers $GTW_WORKERS_NUMBER \
				--ignoreUnresolved \
				--wafName "ModSecurity PARANOIA $PARANOIA" \
				--noEmailReport \
				--includePayloads \
	)

	OVERALL_SCORE="$(echo "$OUTPUT" | grep -E '\| *SCORE *\|' | cut -d '|' -f 3 | sed 's/^ *//; s/%//; s/ *$//')"
	API_SEC="$(echo "$OUTPUT" | grep -E '\| *API Security *\|' | cut -d '|' -f 5 | sed 's/^ *//; s/%//; s/ *$//')"
	APP_SEC="$(echo "$OUTPUT" | grep -E '\| *Application Security *\|' | cut -d '|' -f 5 | sed 's/^ *//; s/%//; s/ *$//')"

	printf "{\n\tName: \"ModSecurity PARANOIA=$PARANOIA\",\n\tApiSec: computeGrade($API_SEC, 1),\n\tAppSec: computeGrade($APP_SEC, 1),\n\tOverallScore: computeGrade($OVERALL_SCORE, 1),\n},\n" >> "$RESULT_OUTPUT"

	docker kill "modsec_paranoia_$PARANOIA"
done

docker kill gotestwaf_test_app
