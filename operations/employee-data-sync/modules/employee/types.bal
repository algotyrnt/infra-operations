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

# [Configurable] OAuth2 application configuration.
type ClientCredentialsOauth2Config record {|
    # OAuth2 token endpoint
    string tokenUrl;
    # OAuth2 client ID
    string clientId;
    # OAuth2 client secret
    string clientSecret;
|};

# Retry config for the graphql client.
public type GraphQlRetryConfig record {|
    # Retry count
    int count = RETRY_COUNT;
    # Retry interval
    decimal interval = RETRY_INTERVAL;
    # Retry backOff factor
    float backOffFactor = RETRY_BACKOFF_FACTOR;
    # Retry max interval
    decimal maxWaitInterval = RETRY_MAX_INTERVAL;
|};

# The EmployeeStatus represents the status of an employee.
public enum EmployeeStatus {
    EmployeeStatusMarkedLeaver = "Marked leaver",
    EmployeeStatusActive = "Active"
}

# Employee filter record.
public type EmployeeFilter record {|
    # Employee employment type
    EmployeeStatus[]? employeeStatus = ();
|};

# Employee type.
public type Employee record {|
    # Work email of the employee
    string workEmail;
    # Thumbnail image of the employee
    string? employeeThumbnail;
    # Job role of the employee
    string? jobRole;
    json...;
|};

# Employee response type.
public type EmployeeResponse record {|
    # Work email of the employee
    string? workEmail;
    # Thumbnail image of the employee
    string? employeeThumbnail;
    # Job role of the employee
    string? jobRole?;
    json...;
|};

# GraphQL multiple employees response.
type MultipleEmployeesResponse record {|
    # Response data wrapper
    record {|
        # Employees data array
        EmployeeResponse[] employees;
    |} data;
|};
