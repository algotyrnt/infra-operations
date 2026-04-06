// Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com). All Rights Reserved.
//
// This software is the property of WSO2 LLC. and its suppliers, if any.
// Dissemination of any information or reproduction of any material contained
// herein in any form is strictly forbidden, unless permitted by WSO2 expressly.
// You may not alter or remove any copyright or other notice from copies of this content.

# Search users from Asgardeo.
#
# + email - User email
# + return - Users result or error
public isolated function searchUser(string email) returns User[]|error {
    UserSearchResult usersResult = check scimOperationsClient->/organizations/internal/users/search.post({
        domain: "DEFAULT",
        filter: string `userName eq ${email}`,
        attributes: ["id", "userName"]
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
