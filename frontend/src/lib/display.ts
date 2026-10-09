// English presentation labels only. Source records and their IDs remain unchanged.
const labels:Record<string,string>={
  'Negosiasi':'Negotiation','Proposal':'Proposal','Discovery':'Discovery','Qualified':'Qualified','Closed Won':'Closed Won','Closed Lost':'Closed Lost',
  'Status pengadaan':'Procurement update','Demo produk Mandala':'Mandala product demo','Proposal KasirNusa untuk 60 gerai':'KasirNusa proposal for 60 stores','Re: Proposal KasirNusa untuk 60 gerai':'Re: KasirNusa proposal for 60 stores','Tindak lanjut demo':'Demo follow-up','Permintaan diskon 20% Teras Kafe':'Teras Kafe: 20% discount request',
  'Bukti sumber':'Source evidence','Rekaman catatan_meeting':'Meeting record','Rekaman email':'Email record','Alamat pada sumber':'Address on the source','Stage historis belum diketahui':'Historical stage unknown',
  'Owner historis belum diketahui':'Historical owner unknown','Rekaman deal':'Deal record','Konteks stakeholder':'Stakeholder context','Interaksi bermakna':'Meaningful interaction','Ketentuan komersial':'Commercial terms','Keputusan / purchase gate':'Decision / purchase gate',
  'Ritel':'Retail','Kesehatan':'Healthcare','Hospitality':'Hospitality','verified':'Verified','inferred':'Inferred','ambiguous':'Ambiguous','unknown':'Unknown',
  'MEETING':'Meeting','FOLLOW_UP':'Follow-up','PROPOSAL':'Proposal','BUYER_REQUEST':'Buyer request','INBOUND_EMAIL':'Inbound email','DISCOUNT_REQUEST':'Discount request','OTHER':'Other'
};
export function display(value:string):string{return labels[value]??value;}
export function dateDisplay(value:string):string{return new Date(value+'T12:00:00').toLocaleDateString('en-GB',{day:'numeric',month:'short',year:'numeric'});}
