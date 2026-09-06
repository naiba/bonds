import type {
  AuthenticationExtensionsClientInputs,
  AuthenticatorTransport,
  PublicKeyCredentialDescriptorJSON,
  PublicKeyCredentialHint,
  PublicKeyCredentialRequestOptionsJSON,
  UserVerificationRequirement,
} from "@simplewebauthn/browser";

class InvalidWebAuthnAuthenticationOptionsError extends Error {
  constructor() {
    super("WebAuthn authentication options response is invalid");
    this.name = "InvalidWebAuthnAuthenticationOptionsError";
  }
}

export function parseWebAuthnAuthenticationOptions(
  responseData: unknown,
): PublicKeyCredentialRequestOptionsJSON {
  if (
    !isRecord(responseData) ||
    !isPublicKeyCredentialRequestOptions(responseData["publicKey"])
  ) {
    throw new InvalidWebAuthnAuthenticationOptionsError();
  }

  return responseData["publicKey"];
}

function isPublicKeyCredentialRequestOptions(
  value: unknown,
): value is PublicKeyCredentialRequestOptionsJSON {
  if (!isRecord(value) || typeof value["challenge"] !== "string") {
    return false;
  }

  return (
    isOptionalNumber(value["timeout"]) &&
    isOptionalString(value["rpId"]) &&
    isOptionalUserVerification(value["userVerification"]) &&
    isOptionalArray(value["allowCredentials"], isCredentialDescriptor) &&
    isOptionalArray(value["hints"], isPublicKeyCredentialHint) &&
    isOptionalExtensions(value["extensions"])
  );
}

function isCredentialDescriptor(
  value: unknown,
): value is PublicKeyCredentialDescriptorJSON {
  return (
    isRecord(value) &&
    typeof value["id"] === "string" &&
    value["type"] === "public-key" &&
    isOptionalArray(value["transports"], isAuthenticatorTransport)
  );
}

function isOptionalExtensions(
  value: unknown,
): value is AuthenticationExtensionsClientInputs | undefined {
  return (
    value === undefined ||
    (isRecord(value) &&
      isOptionalString(value["appid"]) &&
      isOptionalBoolean(value["credProps"]) &&
      isOptionalBoolean(value["hmacCreateSecret"]) &&
      isOptionalBoolean(value["minPinLength"]))
  );
}

function isAuthenticatorTransport(
  value: unknown,
): value is AuthenticatorTransport {
  switch (value) {
    case "ble":
    case "hybrid":
    case "internal":
    case "nfc":
    case "usb":
      return true;
    default:
      return false;
  }
}

function isPublicKeyCredentialHint(
  value: unknown,
): value is PublicKeyCredentialHint {
  switch (value) {
    case "hybrid":
    case "security-key":
    case "client-device":
      return true;
    default:
      return false;
  }
}

function isOptionalUserVerification(
  value: unknown,
): value is UserVerificationRequirement | undefined {
  return (
    value === undefined ||
    value === "discouraged" ||
    value === "preferred" ||
    value === "required"
  );
}

function isOptionalArray<T>(
  value: unknown,
  itemGuard: (item: unknown) => item is T,
): value is readonly T[] | undefined {
  return value === undefined || (Array.isArray(value) && value.every(itemGuard));
}

function isOptionalBoolean(value: unknown): value is boolean | undefined {
  return value === undefined || typeof value === "boolean";
}

function isOptionalNumber(value: unknown): value is number | undefined {
  return value === undefined || typeof value === "number";
}

function isOptionalString(value: unknown): value is string | undefined {
  return value === undefined || typeof value === "string";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
