'use client'

import { useEffect, useState } from 'react'
import { supabase, TableData } from '@/lib/supabase'

export default function Home() {
  const [data, setData] = useState<TableData[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState<Omit<TableData, 'id' | 'created_at'>>({
    fio: '',
    ipn: '',
    organization: '',
    date_opened: '',
    date_first_deposit: '',
    account_status: '',
    card_status: '',
    documents: '',
    comment: ''
  })

  useEffect(() => {
    fetchData()
  }, [])

  const fetchData = async () => {
    try {
      const { data: tableData, error } = await supabase
        .from('zvit_table')
        .select('*')
        .order('id', { ascending: false })

      if (error) {
        console.error('Помилка завантаження даних:', error)
      } else {
        setData(tableData || [])
      }
    } catch (error) {
      console.error('Помилка:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    try {
      const { error } = await supabase
        .from('zvit_table')
        .insert([formData])

      if (error) {
        console.error('Помилка додавання даних:', error)
        alert('Помилка додавання даних')
      } else {
        alert('Дані успішно додано!')
        setFormData({
          fio: '',
          ipn: '',
          organization: '',
          date_opened: '',
          date_first_deposit: '',
          account_status: '',
          card_status: '',
          documents: '',
          comment: ''
        })
        setShowForm(false)
        fetchData()
      }
    } catch (error) {
      console.error('Помилка:', error)
      alert('Помилка при додаванні даних')
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: value
    }))
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl">Завантаження...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen p-4 bg-gray-50">
      <div className="max-w-full mx-auto">
        <div className="bg-white rounded-lg shadow-md">
          <div className="p-6 border-b border-gray-200">
            <div className="flex justify-between items-center">
              <h1 className="text-2xl font-bold text-gray-800">Звітна таблиця</h1>
              <button
                onClick={() => setShowForm(!showForm)}
                className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-lg transition-colors"
              >
                {showForm ? 'Скасувати' : 'Додати запис'}
              </button>
            </div>
          </div>

          {showForm && (
            <div className="p-6 border-b border-gray-200 bg-gray-50">
              <form onSubmit={handleSubmit} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">ФІО</label>
                  <input
                    type="text"
                    name="fio"
                    value={formData.fio}
                    onChange={handleInputChange}
                    required
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">ІПН</label>
                  <input
                    type="text"
                    name="ipn"
                    value={formData.ipn}
                    onChange={handleInputChange}
                    required
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Організація</label>
                  <input
                    type="text"
                    name="organization"
                    value={formData.organization}
                    onChange={handleInputChange}
                    required
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Дата відкриття</label>
                  <input
                    type="date"
                    name="date_opened"
                    value={formData.date_opened}
                    onChange={handleInputChange}
                    required
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Дата першого зарахування</label>
                  <input
                    type="date"
                    name="date_first_deposit"
                    value={formData.date_first_deposit}
                    onChange={handleInputChange}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Статус рахунку</label>
                  <select
                    name="account_status"
                    value={formData.account_status}
                    onChange={handleInputChange}
                    required
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">Оберіть статус</option>
                    <option value="активний">Активний</option>
                    <option value="заблокований">Заблокований</option>
                    <option value="закритий">Закритий</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Статус карти</label>
                  <select
                    name="card_status"
                    value={formData.card_status}
                    onChange={handleInputChange}
                    required
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">Оберіть статус</option>
                    <option value="активна">Активна</option>
                    <option value="заблокована">Заблокована</option>
                    <option value="не випущена">Не випущена</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Документи</label>
                  <input
                    type="text"
                    name="documents"
                    value={formData.documents}
                    onChange={handleInputChange}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div className="md:col-span-2 lg:col-span-3">
                  <label className="block text-sm font-medium text-gray-700 mb-1">Коментар</label>
                  <textarea
                    name="comment"
                    value={formData.comment}
                    onChange={handleInputChange}
                    rows={3}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div className="md:col-span-2 lg:col-span-3">
                  <button
                    type="submit"
                    className="bg-green-500 hover:bg-green-600 text-white px-6 py-2 rounded-lg transition-colors"
                  >
                    Зберегти
                  </button>
                </div>
              </form>
            </div>
          )}

          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-100">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">ФІО</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">ІПН</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">Організація</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">Дата відкриття</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">Дата першого зарахування</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">Статус рахунку</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">Статус карти</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">Документи</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b">Коментар</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {data.length === 0 ? (
                  <tr>
                    <td colSpan={9} className="px-4 py-8 text-center text-gray-500">
                      Дані відсутні. Додайте перший запис.
                    </td>
                  </tr>
                ) : (
                  data.map((row) => (
                    <tr key={row.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm text-gray-900 border-b">{row.fio}</td>
                      <td className="px-4 py-3 text-sm text-gray-900 border-b">{row.ipn}</td>
                      <td className="px-4 py-3 text-sm text-gray-900 border-b">{row.organization}</td>
                      <td className="px-4 py-3 text-sm text-gray-900 border-b">
                        {row.date_opened ? new Date(row.date_opened).toLocaleDateString('uk-UA') : ''}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-900 border-b">
                        {row.date_first_deposit ? new Date(row.date_first_deposit).toLocaleDateString('uk-UA') : ''}
                      </td>
                      <td className="px-4 py-3 text-sm border-b">
                        <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                          row.account_status === 'активний' ? 'bg-green-100 text-green-800' :
                          row.account_status === 'заблокований' ? 'bg-red-100 text-red-800' :
                          'bg-gray-100 text-gray-800'
                        }`}>
                          {row.account_status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm border-b">
                        <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                          row.card_status === 'активна' ? 'bg-green-100 text-green-800' :
                          row.card_status === 'заблокована' ? 'bg-red-100 text-red-800' :
                          'bg-gray-100 text-gray-800'
                        }`}>
                          {row.card_status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-900 border-b max-w-xs truncate" title={row.documents}>
                        {row.documents}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-900 border-b max-w-xs truncate" title={row.comment}>
                        {row.comment}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  )
}
