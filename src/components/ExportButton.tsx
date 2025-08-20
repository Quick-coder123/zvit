'use client'

import { TableData } from '@/lib/supabase'

interface ExportButtonProps {
  data: TableData[]
}

export default function ExportButton({ data }: ExportButtonProps) {
  const exportToCSV = () => {
    if (data.length === 0) {
      alert('Немає даних для експорту')
      return
    }

    // Заголовки
    const headers = [
      'ФІО',
      'ІПН', 
      'Організація',
      'Дата відкриття',
      'Дата першого зарахування',
      'Статус рахунку',
      'Статус карти',
      'Документи',
      'Коментар'
    ]

    // Дані
    const csvData = data.map(row => [
      row.fio,
      row.ipn,
      row.organization,
      row.date_opened,
      row.date_first_deposit || '',
      row.account_status,
      row.card_status,
      row.documents || '',
      row.comment || ''
    ])

    // Об'єднання заголовків та даних
    const csvContent = [headers, ...csvData]
      .map(row => row.map(field => `"${field}"`).join(','))
      .join('\n')

    // Створення та завантаження файлу
    const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    const url = URL.createObjectURL(blob)
    link.setAttribute('href', url)
    link.setAttribute('download', `zvit_${new Date().toISOString().split('T')[0]}.csv`)
    link.style.visibility = 'hidden'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  return (
    <button
      onClick={exportToCSV}
      className="bg-green-600 hover:bg-green-700 text-white px-5 py-2.5 rounded-lg transition-colors flex items-center gap-2 font-medium"
    >
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
      </svg>
      Експорт CSV
    </button>
  )
}
