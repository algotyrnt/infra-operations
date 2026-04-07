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
import ballerina/log;

# Get Employees by filters.
#
# + filters - Filter object containing the filter criteria for the query
# + return - Return an array of Employee entity or error
public isolated function getEmployees(EmployeeFilter filters = {}) returns Employee[]|error {
    string document = string `
        query getEmployees($filter: EmployeeFilter!, $limit: Int, $offset: Int) {
            employees(filter: $filter, limit: $limit, offset: $offset) {
                workEmail
                employeeThumbnail
                jobRole
            }
        }
    `;

    Employee[] employees = [];
    boolean fetchMore = true;
    int offset = 0;
    int defaultLimit = 100;

    while fetchMore {
        MultipleEmployeesResponse|graphql:ClientError response = employeeClient->execute(
            document,
            {filter: filters, 'limit: defaultLimit, offset}
        );
        if response is graphql:ClientError {
            string customError = "An error occurred while retrieving employee data!";
            log:printError(customError, response);
            return error(customError, response);
        }
        EmployeeResponse[] batch = response.data.employees;
        Employee[] batchEmployees = from EmployeeResponse empResp in batch
            select {
                workEmail: empResp.workEmail ?: "",
                employeeThumbnail: empResp.employeeThumbnail,
                jobRole: empResp?.jobRole
            };
        employees.push(...batchEmployees);
        fetchMore = batch.length() > 0;
        offset += defaultLimit;
    }
    return employees;
}
