import { graphql } from "../gql";

export const MeQuery = graphql(`
  query Me {
    me {
      id
      email
      displayName
    }
  }
`);

export const SendPasscodeMutation = graphql(`
  mutation SendPasscode($email: String!) {
    sendPasscode(email: $email) {
      success
    }
  }
`);

export const VerifyPasscodeMutation = graphql(`
  mutation VerifyPasscode($email: String!, $code: String!) {
    verifyPasscode(email: $email, code: $code) {
      accessToken
      refreshToken
      user {
        id
        email
        displayName
      }
    }
  }
`);

export const RefreshTokenMutation = graphql(`
  mutation RefreshToken($refreshToken: String!) {
    refreshToken(refreshToken: $refreshToken) {
      accessToken
      refreshToken
      user {
        id
        email
        displayName
      }
    }
  }
`);

export const GroupsQuery = graphql(`
  query Groups {
    groups {
      id
      name
      createdBy {
        id
        displayName
      }
      members {
        id
        displayName
      }
      createdAt
    }
  }
`);

export const GroupQuery = graphql(`
  query Group($id: ID!) {
    group(id: $id) {
      id
      name
      createdBy {
        id
        displayName
      }
      members {
        id
        email
        displayName
      }
      createdAt
    }
  }
`);

export const CreateGroupMutation = graphql(`
  mutation CreateGroup($input: CreateGroupInput!) {
    createGroup(input: $input) {
      id
      name
    }
  }
`);
