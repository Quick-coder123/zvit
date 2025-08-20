"use client";

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { supabase, TableData } from '@/lib/supabase';
import ExportButton from '@/components/ExportButton';
import ImportExcel from '@/components/ImportExcel';
import Link from 'next/link';

export default function Home() {
  const [isAuthChecked, setIsAuthChecked] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [data, setData] = useState<TableData[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<'checking' | 'connected' | 'error'>('checking');
  const [editingId, setEditingId] = useState<number | null>(null);
  const [formData, setFormData] = useState<Omit<TableData, 'id' | 'created_at' | 'account_status'>>({
    fio: '',
    ipn: '',
    organization: '',
    date_opened: '',
    date_first_deposit: '',
    card_status: 'На випуску',
    documents: {
      contract: false,
      passport: false,
      questionnaire: false,
    },
    comment: '',
  });
  const router = useRouter();

  useEffect(() => {
    async function checkAuth() {
      const { data: { session } } = await supabase.auth.getSession();
      if (!session) {
        router.replace('/admin');
      } else {
        setIsAuthenticated(true);
      }
      setIsAuthChecked(true);
    }
    checkAuth();
  }, [router]);

  useEffect(() => {
    const checkConnection = async () => {
      try {
        const { error } = await supabase.from('zvit_table').select('count', { count: 'exact' }).limit(1);
        if (error) {
          setConnectionStatus('error');
        } else {
          setConnectionStatus('connected');
        }
      } catch {
        setConnectionStatus('error');
      }
    };

    checkConnection();
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const { data: tableData } = await supabase.from('zvit_table').select('*').order('id', { ascending: false });
      setData(tableData || []);
    } catch {
      setData([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
  console.log('Відправляємо дані:', formData);
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Ви впевнені, що хочете видалити цей запис?')) {
      return;
    }

    try {
      const { error } = await supabase
        .from('zvit_table')
        .delete()
        .eq('id', id);

      if (error) {
        console.error('Помилка видалення:', error);
        alert('Помилка видалення запису');
      } else {
        alert('Запис успішно видалено!');
        fetchData();
      }
    } catch (error) {
      console.error('Помилка:', error);
      alert('Помилка при видаленні запису');
    }
  }

  const testConnection = async () => {
    try {
      console.log('Тестуємо підключення...');
      const { data, error } = await supabase
        .from('zvit_table')
        .select('*')
        .limit(1);
      
      if (error) {
        console.error('Помилка тестування:', error);
        alert(`Помилка підключення: ${error.message}`);
      } else {
        console.log('Тест успішний:', data);
        alert('Підключення працює!');
      }
    } catch (error) {
      console.error('Помилка тестування:', error);
      alert('Помилка тестування підключення');
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  }

  const handleEdit = (item: TableData) => {
    if (!item.id) return;
    setEditingId(item.id);
    setFormData({
      fio: item.fio,
      ipn: item.ipn,
      organization: item.organization,
      date_opened: item.date_opened,
      date_first_deposit: item.date_first_deposit || '',
      card_status: item.card_status,
      documents: item.documents,
      comment: item.comment || ''
    });
    setShowForm(true);
  }

  const handleCancelEdit = () => {
    setEditingId(null);
    setFormData({
      fio: '',
      ipn: '',
      organization: '',
      date_opened: '',
      date_first_deposit: '',
      card_status: 'На випуску',
      documents: {
        contract: false,
        passport: false,
        questionnaire: false
      },
      comment: ''
    });
    setShowForm(false);
  }

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!editingId) return;
    
    console.log('Оновлюємо дані:', formData);
    
    try {
      const dataToUpdate = {
        ...formData,
        date_first_deposit: formData.date_first_deposit || null
      };
      
      console.log('Підготовлені дані для оновлення:', dataToUpdate);
      
      const { data, error } = await supabase
        .from('zvit_table')
        .update(dataToUpdate)
        .eq('id', editingId)
        .select();

      if (error) {
        console.error('Помилка оновлення даних:', error);
        console.error('Деталі помилки:', error.message, error.details, error.hint);
        alert(`Помилка оновлення даних: ${error.message}`);
      } else {
        console.log('Дані успішно оновлено:', data);
        alert('Дані успішно оновлено!');
        handleCancelEdit();
        fetchData(); // Оновлюємо список
      }
    } catch (error) {
      console.error('Помилка оновлення:', error);
      alert('Помилка оновлення даних');
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-2xl font-semibold text-gray-800">Завантаження...</div>
      </div>
    );
  }
  if (!isAuthChecked) {
    return <div className="min-h-screen flex items-center justify-center">Перевірка авторизації...</div>;
  }
  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="max-w-full mx-auto bg-gray-50 min-h-screen">
      <header className="bg-blue-700 text-white shadow-md">
        <div className="max-w-7xl mx-auto flex justify-between items-center px-6 py-4">
          <h1 className="text-xl font-bold">Головна сторінка</h1>
          <nav className="flex gap-6">
            <Link href="/" className="hover:underline text-lg">Головна</Link>
            <Link href="/zvit" className="hover:underline text-lg">Звіт</Link>
          </nav>
        </div>
      </header>

      <div className="bg-white rounded-lg shadow-md">
        <div className="p-6 border-b border-gray-200">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Звітна таблиця</h1>
              {connectionStatus === 'checking' && <p className="text-sm text-yellow-600 mt-1">Перевіряємо підключення...</p>}
              {connectionStatus === 'error' && <p className="text-sm text-red-600 mt-1">❌ Помилка підключення до бази даних</p>}
              {connectionStatus === 'connected' && <p className="text-sm text-green-600 mt-1">✅ Підключено до бази даних</p>}
            </div>
            <div className="flex gap-2">
              {connectionStatus === 'error' && (
                <button
                  onClick={() => fetchData()}
                  className="bg-yellow-600 hover:bg-yellow-700 text-white px-4 py-2 rounded-lg transition-colors font-medium"
                >
                  🔧 Тест підключення
                </button>
              )}
              <ExportButton data={data} />
              <button
                onClick={() => setShowImport(!showImport)}
                className="bg-purple-600 hover:bg-purple-700 text-white px-5 py-2.5 rounded-lg transition-colors font-medium"
                disabled={connectionStatus === 'error'}
              >
                {showImport ? 'Скасувати імпорт' : 'Імпорт Excel'}
              </button>
              <button
                onClick={() => setShowForm(!showForm)}
                className="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2.5 rounded-lg transition-colors font-medium"
                disabled={connectionStatus === 'error'}
              >
                {showForm ? 'Скасувати' : 'Додати запис'}
              </button>
            </div>
          </div>
        </div>

        {showImport && (
          <div className="p-6 border-b border-gray-200 bg-purple-50">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Імпорт даних з Excel</h3>
            <ImportExcel onImportComplete={() => fetchData()} />
          </div>
        )}

        {showForm && (
          <div className="p-6 border-b border-gray-200 bg-gray-50">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Додати новий запис</h3>
            <form className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-semibold text-gray-800 mb-2">ФІО</label>
                <input
                  type="text"
                  name="fio"
                  value={formData.fio}
                  onChange={(e) => setFormData({ ...formData, fio: e.target.value })}
                  required
                  className="w-full border border-gray-400 rounded-md px-4 py-3 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-800 mb-2">ІПН</label>
                <input
                  type="text"
                  name="ipn"
                  value={formData.ipn}
                  onChange={(e) => setFormData({ ...formData, ipn: e.target.value })}
                  required
                  className="w-full border border-gray-400 rounded-md px-4 py-3 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-800 mb-2">Організація</label>
                <input
                  type="text"
                  name="organization"
                  value={formData.organization}
                  onChange={(e) => setFormData({ ...formData, organization: e.target.value })}
                  required
                  className="w-full border border-gray-400 rounded-md px-4 py-3 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-800 mb-2">Дата відкриття</label>
                <input
                  type="date"
                  name="date_opened"
                  value={formData.date_opened}
                  onChange={(e) => setFormData({ ...formData, date_opened: e.target.value })}
                  required
                  className="w-full border border-gray-400 rounded-md px-4 py-3 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-800 mb-2">Дата першого зарахування</label>
                <input
                  type="date"
                  name="date_first_deposit"
                  value={formData.date_first_deposit}
                  onChange={(e) => setFormData({ ...formData, date_first_deposit: e.target.value })}
                  className="w-full border border-gray-400 rounded-md px-4 py-3 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                <p className="text-xs text-gray-600 mt-1">
                  💡 Статус рахунку буде автоматично встановлено: 
                  {formData.date_first_deposit ? ' "Активний"' : ' "Очікує активацію"'}
                </p>
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-800 mb-2">Статус карти</label>
                <select
                  name="card_status"
                  value={formData.card_status}
                  onChange={(e) => setFormData({ ...formData, card_status: e.target.value as 'На випуску' | 'На відділенні' | 'На організації' | 'Активована' })}
                  required
                  className="w-full border border-gray-400 rounded-md px-4 py-3 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="">Оберіть статус</option>
                  <option value="На випуску">На випуску</option>
                  <option value="На відділенні">На відділенні</option>
                  <option value="На організації">На організації</option>
                  <option value="Активована">Активована</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-800 mb-2">Документи</label>
                <div className="space-y-2">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={formData.documents.contract}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        documents: {
                          ...prev.documents,
                          contract: e.target.checked
                        }
                      }))}
                      className="mr-2 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    />
                    <span className="text-sm text-gray-900">Договір</span>
                  </label>
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={formData.documents.passport}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        documents: {
                          ...prev.documents,
                          passport: e.target.checked
                        }
                      }))}
                      className="mr-2 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    />
                    <span className="text-sm text-gray-900">Паспорт</span>
                  </label>
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={formData.documents.questionnaire}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        documents: {
                          ...prev.documents,
                          questionnaire: e.target.checked
                        }
                      }))}
                      className="mr-2 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    />
                    <span className="text-sm text-gray-900">Опитувальник</span>
                  </label>
                </div>
              </div>

              <div className="md:col-span-2 lg:col-span-3">
                <label className="block text-sm font-semibold text-gray-800 mb-2">Коментар</label>
                <textarea
                  name="comment"
                  value={formData.comment}
                  onChange={(e) => setFormData({ ...formData, comment: e.target.value })}
                  rows={3}
                  className="w-full border border-gray-400 rounded-md px-4 py-3 text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              <div className="md:col-span-2 lg:col-span-3">
                <button
                  type="submit"
                  className="bg-green-600 hover:bg-green-700 text-white px-8 py-3 rounded-lg transition-colors font-semibold"
                >
                  {editingId ? 'Оновити' : 'Зберегти'}
                </button>
              </div>
            </form>
          </div>
        )}

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-100">
              <tr>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">ФІО</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">ІПН</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Організація</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Дата відкриття</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Дата першого зарахування</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Статус рахунку</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Статус карти</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Документи</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Коментар</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Дії</th>
                <th className="px-4 py-4 text-left text-sm font-bold text-gray-900 uppercase tracking-wider border-b-2 border-gray-300">Дії</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data.length === 0 ? (
                <tr>
                  <td colSpan={11} className="px-4 py-8 text-center text-gray-600 text-lg font-medium">
                    Дані відсутні. Додайте перший запис.
                  </td>
                </tr>
              ) : (
                data.map((row) => (
                  <tr key={row.id} className="hover:bg-gray-50">
                    <td className="px-4 py-4 text-sm font-medium text-gray-900 border-b">{row.fio}</td>
                    <td className="px-4 py-4 text-sm font-medium text-gray-900 border-b">{row.ipn}</td>
                    <td className="px-4 py-4 text-sm font-medium text-gray-900 border-b">{row.organization}</td>
                    <td className="px-4 py-4 text-sm font-medium text-gray-900 border-b">
                      {row.date_opened ? new Date(row.date_opened).toLocaleDateString('uk-UA') : ''}
                    </td>
                    <td className="px-4 py-4 text-sm font-medium text-gray-900 border-b">
                      {row.date_first_deposit ? new Date(row.date_first_deposit).toLocaleDateString('uk-UA') : ''}
                    </td>
                    <td className="px-4 py-4 text-sm border-b">
                      <span className={`inline-flex px-3 py-1 text-sm font-bold rounded-full ${
                        row.account_status === 'Активний' ? 'bg-green-100 text-green-900' :
                        row.account_status === 'Очікує активацію' ? 'bg-yellow-100 text-yellow-900' :
                        row.account_status === 'Заблокований' ? 'bg-red-100 text-red-900' :
                        'bg-gray-100 text-gray-900'
                      }`}>
                        {row.account_status}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-sm border-b">
                      <span className={`inline-flex px-3 py-1 text-sm font-bold rounded-full ${
                        row.card_status === 'Активована' ? 'bg-green-100 text-green-900' :
                        row.card_status === 'На організації' ? 'bg-blue-100 text-blue-900' :
                        row.card_status === 'На відділенні' ? 'bg-yellow-100 text-yellow-900' :
                        row.card_status === 'На випуску' ? 'bg-purple-100 text-purple-900' :
                        'bg-gray-100 text-gray-900'
                      }`}>
                        {row.card_status}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-sm font-medium text-gray-900 border-b max-w-xs">
                      <div className="flex gap-2">
                        <span className={`px-2 py-1 text-xs rounded-full ${row.documents.contract ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-500'}`}>
                          Договір
                        </span>
                        <span className={`px-2 py-1 text-xs rounded-full ${row.documents.passport ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-500'}`}>
                          Паспорт
                        </span>
                        <span className={`px-2 py-1 text-xs rounded-full ${row.documents.questionnaire ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-500'}`}>
                          Опитувальник
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-4 text-sm font-medium text-gray-900 border-b max-w-xs truncate" title={row.comment}>
                      {row.comment}
                    </td>
                    <td className="px-4 py-4 text-sm border-b">
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleEdit(row)}
                          className="text-blue-600 hover:text-blue-800 transition-colors p-1"
                          title="Редагувати запис"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                          </svg>
                        </button>
                        <button
                          onClick={() => handleDelete(row.id!)}
                          className="text-red-700 hover:text-red-900 transition-colors p-1"
                          title="Видалити запис"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}

            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
