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

import employee_data_sync.employee;
import employee_data_sync.scim;

import ballerina/lang.runtime;
import ballerina/log;

@display {
    label: "Employee Data Sync to Asgardeo",
    id: "infra/employee-data-sync"
}
public function main() returns error? {
    employee:Employee[] employees = check employee:getEmployees(filters =
            {employeeStatus: [employee:EmployeeStatusActive, employee:EmployeeStatusMarkedLeaver]});
    log:printInfo("Successfully fetched employee data. Total employees: " + employees.length().toString());
    int count = 1;
    foreach employee:Employee employee in employees {
        if count % 100 == 0 {
            log:printInfo(string `Processed ${count} employees so far...`);
            log:printInfo("Waiting for 1 minute to avoid hitting rate limits...");
            // Wait for 1 minute after processing every 100 employees to avoid hitting rate limits.
            runtime:sleep(60);
        }
        scim:User[] userResult = check scim:searchUser(employee.workEmail);
        if userResult.length() == 0 {
            log:printInfo(string `User with email ${employee.workEmail} does not exist in Asgardeo. Skipping...`);
            count += 1;
            continue;
        }
        scim:User user = userResult[0];
        if user.urn\:scim\:wso2\:schema?.jobtitle is () && user.profileUrl is () {
            scim:User|error updatedUser = scim:updateUser(
                    {jobTitle: employee.jobRole ?: "", profileUrl: employee.employeeThumbnail ?: ""}, user.id);
            if updatedUser is error {
                log:printError(string `Failed to update user: ${user.userName} in Asgardeo.`, updatedUser);
            } else {
                if updatedUser.profileUrl == employee.employeeThumbnail &&
                    updatedUser.urn\:scim\:wso2\:schema?.jobtitle == employee.jobRole {
                    log:printDebug(string `Successfully updated user: ${user.userName} in Asgardeo.`);
                }
            }
        } else if user.urn\:scim\:wso2\:schema?.jobtitle != employee.jobRole {
            scim:User|error updatedUser = scim:updateUser({jobTitle: employee.jobRole ?: ""}, user.id);
            if updatedUser is error {
                log:printError(string `Failed to update job title for user: ${user.userName} in Asgardeo.`,
                        updatedUser);
            } else {
                if updatedUser.urn\:scim\:wso2\:schema?.jobtitle == employee.jobRole {
                    log:printDebug(string `Successfully updated job title for user: ${user.userName} in Asgardeo.`);
                }
            }
        } else if user.profileUrl != employee.employeeThumbnail {
            scim:User|error updatedUser = scim:updateUser({profileUrl: employee.employeeThumbnail ?: ""}, user.id);
            if updatedUser is error {
                log:printError(string `Failed to update profile URL for user: ${user.userName} in Asgardeo.`,
                        updatedUser);
            } else {
                if updatedUser.profileUrl == employee.employeeThumbnail {
                    log:printDebug(string `Successfully updated profile URL for user: ${user.userName} in Asgardeo.`);
                }
            }
        }
        count += 1;
    }
    log:printInfo("Employees data sync to Asgardeo completed successfully.");
}
