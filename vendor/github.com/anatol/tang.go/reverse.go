package tang

import (
	"bufio"
	"net"
)

// ReverseTangHandshake performs a key exchange with "remote" clevis client
func ReverseTangHandshake(address string, ks *KeySet) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Write(ks.DefaultAdvertisement); err != nil {
		return err
	}
	if _, err := conn.Write([]byte("\n")); err != nil {
		return err
	}

	buff := bufio.NewReader(conn)
	t, _, err := buff.ReadLine()
	if err != nil {
		return err
	}
	thp := string(t)
	xchgKey, _, err := buff.ReadLine()
	if err != nil {
		return err
	}

	out, err := ks.Recover(thp, xchgKey)
	if err != nil {
		return err
	}

	_, err = conn.Write(out)
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte("\n"))
	if err != nil {
		return err
	}

	return nil
}
