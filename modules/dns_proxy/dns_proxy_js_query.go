package dns_proxy

import (
	"encoding/json"
	"fmt"

	"github.com/bettercap/bettercap/v2/log"
	"github.com/bettercap/bettercap/v2/session"

	"github.com/miekg/dns"
)

type JSQuery struct {
	Answers     []map[string]interface{}
	Client      map[string]string
	Compress    bool
	Extras      []map[string]interface{}
	Header      JSQueryHeader
	Nameservers []map[string]interface{}
	Questions   []map[string]interface{}

	refHash string
}

type JSQueryHeader struct {
	AuthenticatedData  bool
	Authoritative      bool
	CheckingDisabled   bool
	Id                 uint16
	Opcode             int
	Rcode              int
	RecursionAvailable bool
	RecursionDesired   bool
	Response           bool
	Truncated          bool
	Zero               bool
}

func (j *JSQuery) NewHash() string {
	answers, _ := json.Marshal(j.Answers)
	extras, _ := json.Marshal(j.Extras)
	nameservers, _ := json.Marshal(j.Nameservers)
	questions, _ := json.Marshal(j.Questions)

	headerHash := fmt.Sprintf("%t.%t.%t.%d.%d.%d.%t.%t.%t.%t.%t",
		j.Header.AuthenticatedData,
		j.Header.Authoritative,
		j.Header.CheckingDisabled,
		j.Header.Id,
		j.Header.Opcode,
		j.Header.Rcode,
		j.Header.RecursionAvailable,
		j.Header.RecursionDesired,
		j.Header.Response,
		j.Header.Truncated,
		j.Header.Zero)

	hash := fmt.Sprintf("%s.%s.%t.%s.%s.%s.%s",
		answers,
		j.Client["IP"],
		j.Compress,
		extras,
		headerHash,
		nameservers,
		questions)

	return hash
}

func NewJSQuery(query *dns.Msg, clientIP string) (jsQuery *JSQuery) {
	answers := make([]map[string]interface{}, len(query.Answer))
	extras := make([]map[string]interface{}, len(query.Extra))
	nameservers := make([]map[string]interface{}, len(query.Ns))
	questions := make([]map[string]interface{}, len(query.Question))

	for i, rr := range query.Answer {
		jsRecord, err := NewJSResourceRecord(rr)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		answers[i] = jsRecord
	}

	for i, rr := range query.Extra {
		jsRecord, err := NewJSResourceRecord(rr)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		extras[i] = jsRecord
	}

	for i, rr := range query.Ns {
		jsRecord, err := NewJSResourceRecord(rr)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		nameservers[i] = jsRecord
	}

	for i, question := range query.Question {
		questions[i] = map[string]interface{}{
			"Name":   question.Name,
			"Qtype":  question.Qtype,
			"Qclass": question.Qclass,
		}
	}

	clientMAC := ""
	clientAlias := ""
	if endpoint := session.I.Lan.GetByIp(clientIP); endpoint != nil {
		clientMAC = endpoint.HwAddress
		clientAlias = endpoint.Alias
	}
	client := map[string]string{"IP": clientIP, "MAC": clientMAC, "Alias": clientAlias}

	jsquery := &JSQuery{
		Answers:  answers,
		Client:   client,
		Compress: query.Compress,
		Extras:   extras,
		Header: JSQueryHeader{
			AuthenticatedData:  query.MsgHdr.AuthenticatedData,
			Authoritative:      query.MsgHdr.Authoritative,
			CheckingDisabled:   query.MsgHdr.CheckingDisabled,
			Id:                 query.MsgHdr.Id,
			Opcode:             query.MsgHdr.Opcode,
			Rcode:              query.MsgHdr.Rcode,
			RecursionAvailable: query.MsgHdr.RecursionAvailable,
			RecursionDesired:   query.MsgHdr.RecursionDesired,
			Response:           query.MsgHdr.Response,
			Truncated:          query.MsgHdr.Truncated,
			Zero:               query.MsgHdr.Zero,
		},
		Nameservers: nameservers,
		Questions:   questions,
	}
	jsquery.UpdateHash()

	return jsquery
}

func (j *JSQuery) ToQuery() *dns.Msg {
	var answers []dns.RR
	var extras []dns.RR
	var nameservers []dns.RR
	var questions []dns.Question

	for _, jsRR := range j.Answers {
		rr, err := ToRR(jsRR)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		answers = append(answers, rr)
	}
	for _, jsRR := range j.Extras {
		rr, err := ToRR(jsRR)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		extras = append(extras, rr)
	}
	for _, jsRR := range j.Nameservers {
		rr, err := ToRR(jsRR)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		nameservers = append(nameservers, rr)
	}

	for _, jsQ := range j.Questions {
		questions = append(questions, dns.Question{
			Name:   jsPropToString(jsQ, "Name"),
			Qtype:  jsPropToUint16(jsQ, "Qtype"),
			Qclass: jsPropToUint16(jsQ, "Qclass"),
		})
	}

	query := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:                 j.Header.Id,
			Response:           j.Header.Response,
			Opcode:             j.Header.Opcode,
			Authoritative:      j.Header.Authoritative,
			Truncated:          j.Header.Truncated,
			RecursionDesired:   j.Header.RecursionDesired,
			RecursionAvailable: j.Header.RecursionAvailable,
			Zero:               j.Header.Zero,
			AuthenticatedData:  j.Header.AuthenticatedData,
			CheckingDisabled:   j.Header.CheckingDisabled,
			Rcode:              j.Header.Rcode,
		},
		Compress: j.Compress,
		Question: questions,
		Answer:   answers,
		Ns:       nameservers,
		Extra:    extras,
	}

	return query
}

func (j *JSQuery) UpdateHash() {
	j.refHash = j.NewHash()
}

func (j *JSQuery) WasModified() bool {
	// check if any of the fields has been changed
	return j.NewHash() != j.refHash
}
