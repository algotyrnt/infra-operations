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

# Search users from Asgardeo.
#
# + email - User email
# + return - Users result or error
public isolated function searchUser(string email) returns User[]|error {
    UserSearchResult usersResult = check scimOperationsClient->/organizations/internal/users/search.post({
        domain: "DEFAULT",
        filter: string `userName eq ${email}`,
        attributes: ["id", "userName", "profileUrl", "urn:scim:wso2:schema"]
    });
    return usersResult.Resources;
}

# Updates a user's information in the SCIM operations service.
#
# + payload - The payload containing the user's updated information
# + uuid - Unique identifier of the user to be updated
# + return - The updated User record, or an error if the operation fails
public isolated function updateUser(UserUpdatePayload payload, string uuid) returns User|error {
    return scimOperationsClient->/organizations/internal/users/[uuid].patch(payload);
}
