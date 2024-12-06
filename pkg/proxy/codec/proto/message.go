/*
   Copyright 2024 The olive Authors

   This program is offered under a commercial and under the AGPL license.
   For AGPL licensing, see below.

   AGPL licensing:
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package proto

type Message struct {
	Data []byte
}

func (m *Message) MarshalJSON() ([]byte, error) {
	return m.Data, nil
}

func (m *Message) UnmarshalJSON(data []byte) error {
	m.Data = data
	return nil
}

func (m *Message) ProtoMessage() {}

func (m *Message) Reset() {
	*m = Message{}
}

func (m *Message) String() string {
	return string(m.Data)
}

func (m *Message) Marshal() ([]byte, error) {
	return m.Data, nil
}

func (m *Message) Unmarshal(data []byte) error {
	m.Data = data
	return nil
}

func NewMessage(data []byte) *Message {
	return &Message{data}
}
