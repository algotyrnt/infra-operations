// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/graphql;

configurable string employeeBaseUrl = ?;
configurable GraphQlRetryConfig retryConfig = {
    count: RETRY_COUNT,
    interval: RETRY_INTERVAL,
    backOffFactor: RETRY_BACKOFF_FACTOR,
    maxWaitInterval: RETRY_MAX_INTERVAL
};
configurable ClientCredentialsOauth2Config clientCredentialsOauth2Config = ?;

@display {
    label: "Employee GraphQL Service",
    id: "employee/entity-graphql-service"
}

final graphql:Client employeeClient = check new (employeeBaseUrl, {
    auth: {
        ...clientCredentialsOauth2Config
    },
    retryConfig: {
        ...retryConfig
    }
});
