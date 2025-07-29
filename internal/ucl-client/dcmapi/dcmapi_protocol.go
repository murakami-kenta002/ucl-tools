// SPDX-License-Identifier: Apache-2.0
/**
 * Copyright (c) 2024  Panasonic Automotive Systems, Co., Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dcmapi

import (
	"encoding/json"
	. "ucl-tools/internal/ulog"
)

type DcmApiProtocol struct {
	FormatV1 struct {
		AppName     string `json:"appli_name"`
		CommandType string `json:"command_type"`
	} `json:"format_v1"`
}

func GenerateAppComm(appName string) (string, error) {

	var dp DcmApiProtocol
	dp.FormatV1.AppName = appName
	dp.FormatV1.CommandType = "get_app_command_and_run"

	jsonBytes, err := json.Marshal(dp)
	if err != nil {
		ELog.Println("JSON Marshal error: ", err)
		return "", err
	}
	return string(jsonBytes), err

}

func GenerateStopAppComm(appName string) (string, error) {

	var dp DcmApiProtocol
	dp.FormatV1.AppName = appName

	jsonBytes, err := json.Marshal(dp)
	if err != nil {
		ELog.Println("JSON Marshal error: ", err)
		return "", err
	}
	return string(jsonBytes), err

}

func GenerateLaunchCompsitorsComm() (string, error) {

	var dp DcmApiProtocol
	dp.FormatV1.AppName = "manageCompositors"
	dp.FormatV1.CommandType = "get_app_command_and_run"

	jsonBytes, err := json.Marshal(dp)
	if err != nil {
		ELog.Println("JSON Marshal error: ", err)
		return "", err
	}
	return string(jsonBytes), err
}

func GenerateStopCompsitorsComm() (string, error) {

	var dp DcmApiProtocol
	dp.FormatV1.AppName = "manageCompositors"

	jsonBytes, err := json.Marshal(dp)
	if err != nil {
		ELog.Println("JSON Marshal error: ", err)
		return "", err
	}
	return string(jsonBytes), err
}
