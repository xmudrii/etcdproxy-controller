#!/usr/bin/env bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

read -r -d '' EXPECTED <<EOF
/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
EOF

STATUS=0
FILES=$(find . -name "*.go" -not -path "./vendor/*")

for FILE in $FILES; do
        if [ "$FILE" == "./pkg/apis/etcd/v1alpha1/zz_generated.deepcopy.go" ]; then
            continue
        fi
        if [ "$FILE" == "./pkg/signals/signal_posix.go" ]; then
            continue
        fi
        HEADER=$(head -n 15 $FILE)
        if [ "$HEADER" != "$EXPECTED" ]; then
                echo "incorrect license header: $FILE"
		STATUS=1
        fi
done

exit $STATUS
