# vCard codec compatibility patches

Base: github.com/emersion/go-vcard v0.1.0 (MIT; original LICENSE retained).
All original production and test files are included. Only card.go, decoder.go,
encoder.go, encoder_test.go and the added organization.go/organization_test.go differ from that base.

The original decoder unescapes backslashes before a caller can distinguish a
literal semicolon from a component separator. The encoder also lacks component
escaping. Application-only splitting cannot recover that lost information.

This patch preserves decoded ORG components in private Field metadata while
keeping Value's existing text representation. Field.Organization and
NewOrganization expose unescaped components; the encoder escapes each component
individually. No protocol-visible
private marker or runtime monkey patch is used.

The encoder also preserves commas in URI values, using explicit VALUE first and
then the property default for the card version (RFC 6350 for v4; SOURCE/URL
and RFC 4770 IMPP for v3). In particular, v4 PHOTO defaults to URI even without
VALUE=uri, while v3 PHOTO retains its existing binary encoding. Explicit
VALUE=text overrides a URI default. Applying TEXT
escaping to a data URI produces `data:image/png;base64\,` and breaks vCard 4.0
photos in clients that follow RFC 6350's URI value grammar. CR, LF and backslash
are percent-encoded to avoid introducing content lines. Other TEXT values retain
their original escaping. This is a codec correction, not a runtime monkey patch
or a separate CardDAV serialization layer.

Remove this local replacement when upstream provides equivalent component-aware
ORG decoding/encoding and URI-value encoding. Keep the organization codec, import/subscription and
HTTP regression tests when upgrading. Run this module's upstream and added tests
with `go test ./...`, as well as the application's dual-database vCard/DAV suites.
