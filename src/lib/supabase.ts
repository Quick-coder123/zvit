import { createClient } from '@supabase/supabase-js'

const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL!
const supabaseAnonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!

export const supabase = createClient(supabaseUrl, supabaseAnonKey)

// Тип для даних таблиці
export interface TableData {
  id?: number
  fio: string
  ipn: string
  organization: string
  date_opened: string
  date_first_deposit: string
  account_status: 'Активний' | 'Очікує активацію' | 'Заблокований' | 'Закритий'
  card_status: 'На випуску' | 'На відділенні' | 'На організації' | 'Активована'
  documents: {
    contract: boolean
    passport: boolean
    questionnaire: boolean
  }
  comment: string
  created_at?: string
}
