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
