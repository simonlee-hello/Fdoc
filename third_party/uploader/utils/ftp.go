package utils

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// FTPUpload uploads a file via anonymous-style FTP STOR.
func FTPUpload(addr, user, pass, remoteName string, r io.Reader) error {
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Minute))
	br := bufio.NewReader(conn)

	if _, err := readFTP(br); err != nil {
		return err
	}
	if err := ftpCmd(conn, br, "USER "+user, 331); err != nil {
		return err
	}
	if err := ftpCmd(conn, br, "PASS "+pass, 230); err != nil {
		return err
	}
	if err := writeFTP(conn, "TYPE I"); err != nil {
		return err
	}
	if _, err := readFTP(br); err != nil {
		return err
	}
	if err := writeFTP(conn, "PASV"); err != nil {
		return err
	}
	pasvLine, err := readFTP(br)
	if err != nil {
		return err
	}
	host, port, err := parsePASV(pasvLine)
	if err != nil {
		return err
	}
	data, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 30*time.Second)
	if err != nil {
		return err
	}
	defer data.Close()

	if err := writeFTP(conn, "STOR "+remoteName); err != nil {
		return err
	}
	codeLine, err := readFTP(br)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(codeLine, "150") && !strings.HasPrefix(codeLine, "125") {
		return fmt.Errorf("ftp STOR: %s", strings.TrimSpace(codeLine))
	}
	if _, err := io.Copy(data, r); err != nil {
		return err
	}
	_ = data.Close()
	done, err := readFTP(br)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(done, "226") && !strings.HasPrefix(done, "250") {
		return fmt.Errorf("ftp transfer: %s", strings.TrimSpace(done))
	}
	_ = writeFTP(conn, "QUIT")
	return nil
}

func writeFTP(conn net.Conn, line string) error {
	_, err := io.WriteString(conn, line+"\r\n")
	return err
}

func readFTP(br *bufio.Reader) (string, error) {
	var lines []string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		if len(line) >= 4 && line[3] == ' ' {
			return strings.Join(lines, "\n"), nil
		}
		if len(line) < 4 {
			return strings.Join(lines, "\n"), nil
		}
	}
}

func ftpCmd(conn net.Conn, br *bufio.Reader, cmd string, wantPrefix int) error {
	if err := writeFTP(conn, cmd); err != nil {
		return err
	}
	resp, err := readFTP(br)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("%d", wantPrefix)
	if !strings.HasPrefix(resp, prefix) {
		return fmt.Errorf("ftp %s: %s", cmd, strings.TrimSpace(resp))
	}
	return nil
}

func parsePASV(line string) (string, int, error) {
	start := strings.Index(line, "(")
	end := strings.Index(line, ")")
	if start < 0 || end <= start {
		return "", 0, fmt.Errorf("bad PASV: %s", line)
	}
	parts := strings.Split(line[start+1:end], ",")
	if len(parts) < 6 {
		return "", 0, fmt.Errorf("bad PASV: %s", line)
	}
	var nums [6]int
	for i := 0; i < 6; i++ {
		n, err := fmt.Sscanf(strings.TrimSpace(parts[i]), "%d", &nums[i])
		if err != nil || n != 1 {
			return "", 0, fmt.Errorf("bad PASV: %s", line)
		}
	}
	host := fmt.Sprintf("%d.%d.%d.%d", nums[0], nums[1], nums[2], nums[3])
	port := nums[4]*256 + nums[5]
	return host, port, nil
}
