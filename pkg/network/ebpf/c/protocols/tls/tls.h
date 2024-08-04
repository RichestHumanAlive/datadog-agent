#ifndef __TLS_H
#define __TLS_H

#include "ktypes.h"
#include "bpf_builtins.h"

#define SSL_VERSION20 0x0200
#define SSL_VERSION30 0x0300
#define TLS_VERSION10 0x0301
#define TLS_VERSION11 0x0302
#define TLS_VERSION12 0x0303
#define TLS_VERSION13 0x0304

#define TLS_HANDSHAKE 0x16
#define TLS_APPLICATION_DATA 0x17

/* https://www.rfc-editor.org/rfc/rfc5246#page-19 6.2. Record Layer */

#define TLS_MAX_PAYLOAD_LENGTH (1 << 14)

// TLS record layer header structure
typedef struct {
    __u8 content_type;
    __u16 version;
    __u16 length;
} __attribute__((packed)) tls_record_header_t;

typedef struct {
    __u8 handshake_type;
    __u32 length : 24;
    __u16 version;
} __attribute__((packed)) tls_hello_message_t;

#define TLS_HANDSHAKE_CLIENT_HELLO 0x01
#define TLS_HANDSHAKE_SERVER_HELLO 0x02

// is_valid_tls_version checks if the given version is a valid TLS version as
// defined in the TLS specification.
static __always_inline bool is_valid_tls_version(__u16 version) {
    switch (version) {
    case SSL_VERSION20:
    case SSL_VERSION30:
    case TLS_VERSION10:
    case TLS_VERSION11:
    case TLS_VERSION12:
    case TLS_VERSION13:
        return true;
    }

    return false;
}

// is_valid_tls_app_data checks if the buffer is a valid TLS Application Data
// record header. The record header is considered valid if:
// - the TLS version field is a known SSL/TLS version
// - the payload length is below the maximum payload length defined in the
//   standard.
// - the payload length + the size of the record header is less than the size
//   of the skb
static __always_inline bool is_valid_tls_app_data(tls_record_header_t *hdr, __u32 buf_size, __u32 skb_len) {
    if (payload_len > TLS_MAX_PAYLOAD_LENGTH) {
        return false;
    }

    return sizeof(*hdr) + payload_len <= skb_len;
}

// is_tls_handshake checks if the given TLS message header is a valid TLS
// handshake message. The message is considered valid if:
// - The type matches CLIENT_HELLO or SERVER_HELLO
// - The version is a known SSL/TLS version
static __always_inline bool is_tls_handshake(tls_hello_message_t *msg) {
    switch (msg->handshake_type) {
    case TLS_HANDSHAKE_CLIENT_HELLO:
    case TLS_HANDSHAKE_SERVER_HELLO:
        return true;
    }

    return false;
}

// is_tls checks if the given buffer is a valid TLS record header. We are
// currently checking for two types of record headers:
// - TLS Handshake record headers
// - TLS Application Data record headers
static __always_inline bool is_tls(const char *buf, __u32 buf_size, __u32 skb_len) {
    if (buf_size < (sizeof(tls_record_header_t) + sizeof(tls_hello_message_t))) {
        return false;
    }

    // Getting TLS record header.
    tls_record_header_t *tls_record_header = (tls_record_header_t *)buf;
    // Converting the fields to host byte order.
    tls_record_header->version = bpf_ntohs(tls_record_header->version);
    tls_record_header->length = bpf_ntohs(tls_record_header->length);

    // Checking the version in the record header.
    if (!is_valid_tls_version(bpf_ntohs(hdr->version))) {
        return false;
    }
    switch (tls_record_header->content_type) {
    case TLS_HANDSHAKE:
        // Checking if the buffer is large enough to contain the handshake.
        if (buf_size < (sizeof(tls_record_header_t) + sizeof(tls_hello_message_t))) {
            return false;
        }
        tls_hello_message_t *tls_hello_message = (tls_hello_message_t *)(buf + sizeof(tls_record_header_t));
        if (!is_tls_handshake(tls_hello_message)) {
            return false;
        }
        tls_hello_message->length = bpf_ntohl(tls_hello_message->length);
        tls_hello_message->version = bpf_ntohs(tls_hello_message->version);
        // The version in the handshake message should be greater than or equal to the version in the record header.
        // The version in the handshake message should be a valid TLS version.
        return is_valid_tls_version(tls_hello_message->version) &&
            tls_hello_message->version >= tls_record_header->version;
    case TLS_APPLICATION_DATA:
        return is_valid_tls_app_data(tls_record_header, buf_size, skb_len);
    }

    return false;
}

#endif
