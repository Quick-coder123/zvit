export interface TableData {
  id?: number
  fio: string
  ipn: string
  organization: string
  date_opened: string
  date_first_deposit: string
  account_status: 'активний' | 'заблокований' | 'закритий'
  card_status: 'активна' | 'заблокована' | 'не випущена'
  documents: string
  comment: string
  created_at?: string
}
