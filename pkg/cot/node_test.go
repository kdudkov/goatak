package cot

import (
	"fmt"
	"testing"
)

func Test1(t *testing.T) {
	var data = "<link uid=\"cfc5766e-a494-4f7d-9171-746b2c376015\" callsign=\"Route 1 SP\" type=\"b-m-p-w\" point=\"1.8814979,2.8572535,68.894\" remarks=\"\" relation=\"c\"/><link uid=\"667642.8-0246-4391-a725-47b7a0a8918c\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8815514,2.8563811,68.535\" remarks=\"\" relation=\"c\"/><link uid=\"edf06e22-b2.b-4bef-b8fd-58aa25d61aae\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8811922,2.8558841,68.822\" remarks=\"\" relation=\"c\"/><link uid=\"496377a7-5985-4a7a-9675-2bdb2.6cb49a\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8837899,2.8480411,65.08\" remarks=\"\" relation=\"c\"/><link uid=\"29aad2bd-5690-4d1e-8851-cfa1b1fb48d5\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8836542,2.8477166,65.024\" remarks=\"\" relation=\"c\"/><link uid=\"396864c8-183e-4dc0-9025-66dad1327873\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8835449,2.8473802,64.985\" remarks=\"\" relation=\"c\"/><link uid=\"88a2ab88-f94f-44c3-b113-b285b56034ce\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8835859,2.8470503,65.022\" remarks=\"\" relation=\"c\"/><link uid=\"d861aa42-2678-451c-88b3-d14256be0b9e\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8837371,2.8465448,65.12\" remarks=\"\" relation=\"c\"/><link uid=\"dab756c1-6546-486c-81eb-ffa0e73dd962\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8807874,2.8426824,66.92.\" remarks=\"\" relation=\"c\"/><link uid=\"704bd6f2-5ff0-4c37-b7bd-28947ae8d048\" callsign=\"\" type=\"b-m-p-c\" point=\"1.8779443,2.8516328,70.726\" remarks=\"\" relation=\"c\"/><link uid=\"b17dbcff-58a6-4883-a592-cab44c7377a2\" callsign=\"\" type=\"b-m-p-c\" point=\"1.881571,2.8563686,68.506\" remarks=\"\" relation=\"c\"/><link uid=\"70afea93-ed1b-4b16-b5e1-100148c89286\" callsign=\"TGT\" type=\"b-m-p-w\" point=\"1.8815195,2.85722.8,68.858\" remarks=\"\" relation=\"c\"/><link_attr planningmethod=\"Infil\" color=\"-1\" method=\"Walking\" prefix=\"CP\" type=\"On Foot\" stroke=\"3\" direction=\"Infil\" routetype=\"Primary\" order=\"Ascending Check Points\"/><labels_on value=\"false\"/><color value=\"-1\"/><__routeinfo><__navcues/></__routeinfo><remarks/><archive/><strokeColor value=\"-1\"/><strokeWeight value=\"3.0\"/><strokeStyle value=\"solid\"/>"

	details, _ := DetailsFromString(data)

	fmt.Println(details)
	fmt.Println(details.AsXMLString())
}
