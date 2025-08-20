'use client'

import { useState } from 'react'
import * as XLSX from 'xlsx'
import { supabase, TableData } from '@/lib/supabase'

interface ImportExcelProps {
  onImportComplete: () => void
}

export default function ImportExcel({ onImportComplete }: ImportExcelProps) {
  const [isImporting, setIsImporting] = useState(false)
  const [importResults, setImportResults] = useState<{
    total: number
    success: number
    errors: string[]
  } | null>(null)

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    setIsImporting(true)
    setImportResults(null)

    try {
      const data = await file.arrayBuffer()
      const workbook = XLSX.read(data)
      const sheetName = workbook.SheetNames[0]
      const worksheet = workbook.Sheets[sheetName]
      const jsonData = XLSX.utils.sheet_to_json(worksheet)

      console.log('Дані з Excel:', jsonData)

      const results = {
        total: jsonData.length,
        success: 0,
        errors: [] as string[]
      }

      for (let i = 0; i < jsonData.length; i++) {
        const row = jsonData[i] as any
        
        try {
          // Перетворюємо дані з Excel у формат нашої бази
          const record = {
            fio: row['ФІО'] || row['fio'] || '',
            ipn: row['ІПН'] || row['ipn'] || '',
            organization: row['ОРГАНІЗАЦІЯ'] || row['organization'] || '',
            date_opened: formatDate(row['ДАТА ВІДКРИТТЯ'] || row['date_opened']),
            date_first_deposit: formatDate(row['ДАТА ПЕРШОГО ЗАРАХУВАННЯ'] || row['date_first_deposit']) || null,
            card_status: row['СТАТУС КАРТИ'] || row['card_status'] || 'На випуску',
            documents: {
              contract: parseYesNo(row['ДОГОВІР'] || row['contract']),
              passport: parseYesNo(row['ПАСПОРТ'] || row['passport']),
              questionnaire: parseYesNo(row['ОПИТУВАЛЬНИК'] || row['questionnaire'])
            },
            comment: row['КОМЕНТАР'] || row['comment'] || ''
          }

          // Валідація обов'язкових полів
          if (!record.fio || !record.ipn || !record.organization || !record.date_opened) {
            results.errors.push(`Рядок ${i + 1}: Відсутні обов'язкові поля (ФІО, ІПН, ОРГАНІЗАЦІЯ, ДАТА ВІДКРИТТЯ)`)
            continue
          }

          console.log(`Додаємо запис ${i + 1}:`, record)

          const { error } = await supabase
            .from('zvit_table')
            .insert([record])

          if (error) {
            console.error(`Помилка вставки рядка ${i + 1}:`, error)
            results.errors.push(`Рядок ${i + 1}: ${error.message}`)
          } else {
            results.success++
          }
        } catch (error) {
          console.error(`Помилка обробки рядка ${i + 1}:`, error)
          results.errors.push(`Рядок ${i + 1}: Помилка обробки даних`)
        }
      }

      setImportResults(results)
      
      if (results.success > 0) {
        onImportComplete()
      }

    } catch (error) {
      console.error('Помилка читання файлу:', error)
      setImportResults({
        total: 0,
        success: 0,
        errors: ['Помилка читання файлу Excel']
      })
    } finally {
      setIsImporting(false)
      // Очищаємо input після обробки
      event.target.value = ''
    }
  }

  const formatDate = (dateValue: any): string => {
    if (!dateValue) return ''
    
    // Якщо це Excel дата (число)
    if (typeof dateValue === 'number') {
      const date = XLSX.SSF.parse_date_code(dateValue)
      return `${date.y}-${String(date.m).padStart(2, '0')}-${String(date.d).padStart(2, '0')}`
    }
    
    // Якщо це рядок дати
    if (typeof dateValue === 'string') {
      const trimmed = dateValue.trim()
      
      // Український формат: день.місяць.рік (наприклад: 15.01.2024)
      const ukrainianFormat = /^(\d{1,2})\.(\d{1,2})\.(\d{4})$/.exec(trimmed)
      if (ukrainianFormat) {
        const [, day, month, year] = ukrainianFormat
        return `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`
      }
      
      // Стандартний ISO формат або інші формати
      const date = new Date(trimmed)
      if (!isNaN(date.getTime())) {
        return date.toISOString().split('T')[0]
      }
    }
    
    return ''
  }

  const parseYesNo = (value: any): boolean => {
    if (typeof value === 'boolean') return value
    if (typeof value === 'string') {
      const lower = value.toLowerCase().trim()
      return lower === 'так' || lower === 'yes' || lower === 'true' || lower === '1' || lower === 'да'
    }
    if (typeof value === 'number') {
      return value === 1
    }
    return false
  }

  const downloadTemplate = () => {
    const templateData = [
      {
        'ФІО': 'Іванов Іван Іванович',
        'ІПН': '1234567890',
        'ОРГАНІЗАЦІЯ': 'ТОВ "ТЕСТ"',
        'ДАТА ВІДКРИТТЯ': '15.01.2024',
        'ДАТА ПЕРШОГО ЗАРАХУВАННЯ': '20.01.2024',
        'СТАТУС КАРТИ': 'На випуску',
        'ДОГОВІР': 'так',
        'ПАСПОРТ': 'так',
        'ОПИТУВАЛЬНИК': 'ні',
        'КОМЕНТАР': 'Приклад запису'
      }
    ]

    const ws = XLSX.utils.json_to_sheet(templateData)
    const wb = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(wb, ws, 'Шаблон')
    XLSX.writeFile(wb, 'shablon_import.xlsx')
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-4 items-center">
        <div>
          <input
            type="file"
            accept=".xlsx,.xls"
            onChange={handleFileUpload}
            disabled={isImporting}
            className="hidden"
            id="excel-upload"
          />
          <label
            htmlFor="excel-upload"
            className={`cursor-pointer inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white ${
              isImporting 
                ? 'bg-gray-400 cursor-not-allowed' 
                : 'bg-green-600 hover:bg-green-700'
            } transition-colors`}
          >
            {isImporting ? (
              <>
                <svg className="animate-spin -ml-1 mr-3 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Імпортування...
              </>
            ) : (
              <>
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                </svg>
                Імпорт з Excel
              </>
            )}
          </label>
        </div>

        <button
          onClick={downloadTemplate}
          className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 transition-colors"
        >
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          Завантажити шаблон
        </button>
      </div>

      {importResults && (
        <div className="bg-gray-50 p-4 rounded-lg">
          <h4 className="font-semibold text-gray-900 mb-2">Результати імпорту:</h4>
          <div className="space-y-1 text-sm">
            <p><span className="font-medium">Всього записів:</span> {importResults.total}</p>
            <p><span className="font-medium text-green-600">Успішно імпортовано:</span> {importResults.success}</p>
            <p><span className="font-medium text-red-600">Помилок:</span> {importResults.errors.length}</p>
          </div>
          
          {importResults.errors.length > 0 && (
            <div className="mt-3">
              <p className="font-medium text-red-600 mb-1">Помилки:</p>
              <div className="max-h-32 overflow-y-auto bg-red-50 p-2 rounded text-xs">
                {importResults.errors.map((error, index) => (
                  <div key={index} className="text-red-700">{error}</div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      <div className="text-xs text-gray-500">
        <p><strong>Формат файлу:</strong> Excel (.xlsx, .xls)</p>
        <p><strong>Обов'язкові колонки:</strong> ФІО, ІПН, ОРГАНІЗАЦІЯ, ДАТА ВІДКРИТТЯ</p>
        <p><strong>Додаткові колонки:</strong> ДАТА ПЕРШОГО ЗАРАХУВАННЯ, СТАТУС КАРТИ, ДОГОВІР, ПАСПОРТ, ОПИТУВАЛЬНИК, КОМЕНТАР</p>
        <p><strong>Формат дат:</strong> день.місяць.рік (наприклад: 15.01.2024) або стандартний формат</p>
        <p><strong>Формат документів:</strong> "так" або "ні" для кожного типу документа</p>
      </div>
    </div>
  )
}
