import base64, urllib.request, ssl

# DNS query for example.com A record
q = bytes([0xAA,0xAA,0x01,0x00,0x00,0x01,0x00,0x00,0x00,0x00,0x00,0x00,
           0x07,0x65,0x78,0x61,0x6D,0x70,0x6C,0x65,0x03,0x63,0x6F,0x6D,
           0x00,0x00,0x01,0x00,0x01])

dns_b64 = base64.urlsafe_b64encode(q).rstrip(b'=').decode()
url = f'https://doh.lllllyccc.qzz.io/dns-query?dns={dns_b64}'

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

req = urllib.request.Request(url, headers={'Accept': 'application/dns-message'})
resp = urllib.request.urlopen(req, context=ctx)
data = resp.read()

print(f'Status: {resp.status}, Response length: {len(data)} bytes')

# Parse the response
import struct
# Skip header (12 bytes), parse question and answer
flags = struct.unpack('!H', data[2:4])[0]
rcode = flags & 0x000F
ancount = struct.unpack('!H', data[6:8])[0]
print(f'Rcode: {rcode}, Answer count: {ancount}')

# Simple IP extraction from answer
if ancount > 0:
    # Skip header + question
    offset = 12
    # Skip question name
    while data[offset] != 0:
        offset += data[offset] + 1
    offset += 5  # null + type(2) + class(2)
    
    for i in range(ancount):
        if data[offset] & 0xC0 == 0xC0:
            offset += 2
        else:
            while data[offset] != 0:
                offset += data[offset] + 1
            offset += 1
        atype = struct.unpack('!H', data[offset:offset+2])[0]
        rdlength = struct.unpack('!H', data[offset+8:offset+10])[0]
        if atype == 1:  # A record
            ip = '.'.join(str(b) for b in data[offset+10:offset+10+rdlength])
            print(f'Resolved IP: {ip}')
        offset += 10 + rdlength
