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

import employee_claims_sync.employee;
import employee_claims_sync.scim;

import ballerina/lang.runtime;
import ballerina/log;

@display {
    label: "Employee Claims Sync to Asgardeo",
    id: "infra/employee-claims-sync"
}
public function main() returns error? {
    log:printInfo("Employee claims sync to Asgardeo started...");
    employee:Employee[] employees = check employee:getEmployees(filters =
            {employeeStatus: [employee:EmployeeStatusActive, employee:EmployeeStatusMarkedLeaver]});
    log:printInfo("Successfully fetched employee data. Total employees: " + employees.length().toString());
    int count = 0;
    int updateFailureCount = 0;
    foreach employee:Employee employee in employees {
        if count > 0 && count % 100 == 0 {
            log:printInfo(string `Processed ${count} employees so far...`);
            log:printInfo("Waiting for 1 minute to avoid hitting rate limits...");
            // Wait for 1 minute after processing every 100 employees to avoid hitting rate limits.
            runtime:sleep(60);
        }
        scim:User[] userResult = check scim:searchUser(employee.workEmail.toLowerAscii());
        if userResult.length() == 0 {
            log:printWarn(string `User with email ${employee.workEmail} does not exist in Asgardeo. Skipping...`);
            count += 1;
            continue;
        }
        scim:User user = userResult[0];
        boolean jobTitleNeedsUpdate = user.urn\:scim\:wso2\:schema?.jobtitle != employee.jobRole;
        boolean profileUrlNeedsUpdate = user.profileUrl != employee.employeeThumbnail;

        if jobTitleNeedsUpdate || profileUrlNeedsUpdate {
            scim:UserUpdatePayload updatePayload = {};
            if jobTitleNeedsUpdate {
                updatePayload.jobTitle = employee.jobRole ?: "";
            }
            if profileUrlNeedsUpdate {
                updatePayload.profileUrl = employee.employeeThumbnail ?: "";
            }
            scim:User|error updatedUser = scim:updateUser(updatePayload, user.id);
            if updatedUser is error {
                log:printError(string `Failed to update user: ${user.userName} in Asgardeo.`, updatedUser);
                updateFailureCount += 1;
            } else {
                log:printDebug(string `Successfully updated user: ${user.userName} in Asgardeo.`);
            }
        }
        count += 1;
    }

    if updateFailureCount == 0 {
        log:printInfo("Employee claims sync completed successfully.");
    } else {
        log:printInfo(string `Employee claims sync completed with ${updateFailureCount} update failure(s).`);
    }
}
