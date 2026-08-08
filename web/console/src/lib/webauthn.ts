/**
 * WebAuthn (FIDO2 / passkey) browser helpers.
 *
 * The backend speaks the standard WebAuthn JSON envelope where every binary
 * field is base64url-encoded. The browser `navigator.credentials` API works in
 * ArrayBuffers, so these helpers convert between the two and serialize the
 * ceremony results into the shape the backend's go-webauthn parser expects.
 */

export function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, "+").replace(/_/g, "/")
  const padding = "=".repeat((4 - (base64.length % 4)) % 4)
  const binary = atob(base64 + padding)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes.buffer
}

export function arrayBufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ""
  for (let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i])
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")
}

/** Serializes a registration (attestation) credential for /webauthn/register/finish. */
export function serializeCredential(cred: PublicKeyCredential) {
  const response = cred.response as AuthenticatorAttestationResponse
  return {
    id: cred.id,
    rawId: arrayBufferToBase64url(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: arrayBufferToBase64url(response.clientDataJSON),
      attestationObject: arrayBufferToBase64url(response.attestationObject),
    },
  }
}

/** Serializes an authentication (assertion) credential for /step-up/verify. */
export function serializeAssertion(cred: PublicKeyCredential) {
  const response = cred.response as AuthenticatorAssertionResponse
  return {
    id: cred.id,
    rawId: arrayBufferToBase64url(cred.rawId),
    type: cred.type,
    response: {
      authenticatorData: arrayBufferToBase64url(response.authenticatorData),
      clientDataJSON: arrayBufferToBase64url(response.clientDataJSON),
      signature: arrayBufferToBase64url(response.signature),
      userHandle: response.userHandle ? arrayBufferToBase64url(response.userHandle) : undefined,
    },
  }
}

/** Shape returned by /mfa/webauthn/auth/begin (challenge + allowed credentials). */
export interface WebAuthnAssertionOptions {
  publicKey: {
    challenge: string
    timeout?: number
    rpId?: string
    userVerification?: UserVerificationRequirement
    allowCredentials?: Array<{
      type: "public-key"
      id: string
      transports?: AuthenticatorTransport[]
    }>
  }
}

/**
 * Runs the browser assertion ceremony for a server-issued challenge and returns
 * the serialized assertion ready to POST. Throws a friendly error if the user
 * dismisses the platform prompt.
 */
export async function getAssertion(options: WebAuthnAssertionOptions) {
  const pk = options.publicKey
  try {
    const credential = await navigator.credentials.get({
      publicKey: {
        ...pk,
        challenge: base64urlToBuffer(pk.challenge),
        allowCredentials: pk.allowCredentials?.map((c) => ({
          ...c,
          id: base64urlToBuffer(c.id),
        })),
      } as PublicKeyCredentialRequestOptions,
    })
    if (!credential) throw new Error("Passkey verification was cancelled")
    return serializeAssertion(credential as PublicKeyCredential)
  } catch (err) {
    if (err instanceof DOMException && err.name === "NotAllowedError") {
      throw new Error("Passkey verification was cancelled")
    }
    throw err
  }
}
