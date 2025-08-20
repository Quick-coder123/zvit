"use client";

import { useEffect, useState } from 'react'
import { supabase } from '@/lib/supabase'
import Link from 'next/link';

interface ReportRow {
  organization: string;
  [month: string]: number | string;
}

const monthNames = [
  'Січень', 'Лютий', 'Березень', 'Квітень', 'Травень', 'Червень',
  'Липень', 'Серпень', 'Вересень', 'Жовтень', 'Листопад', 'Грудень'
]

export default function ZvitPage() {
  // --- Додатковий звіт: організація × місяць, рахунки активовані за датою першого зарахування ---
  const [activated, setActivated] = useState<{ orgs: string[], orgMonthMap: Record<string, number[]> }>({ orgs: [], orgMonthMap: {} });
  useEffect(() => {
    async function fetchActivated() {
      const { data, error } = await supabase
        .from('zvit_table')
        .select('organization, date_first_deposit')
        .not('date_first_deposit', 'is', null);
      const orgMonthMap: Record<string, number[]> = {};
      (data || []).forEach(row => {
        if (row.organization && row.date_first_deposit) {
          const org = row.organization;
          const date = new Date(row.date_first_deposit);
          if (!isNaN(date.getTime())) {
            if (!orgMonthMap[org]) orgMonthMap[org] = Array(12).fill(0);
            orgMonthMap[org][date.getMonth()]++;
          }
        }
      });
      setActivated({ orgs: Object.keys(orgMonthMap), orgMonthMap });
    }
    fetchActivated();
  }, []);

  const [report, setReport] = useState<ReportRow[]>([])
  const [loading, setLoading] = useState(true)
  const [year, setYear] = useState<number>(new Date().getFullYear())

  useEffect(() => {
    fetchReport(year)
  }, [year])

  async function fetchReport(selectedYear: number) {
    setLoading(true)
    // Отримуємо всі рахунки за рік
    const { data, error } = await supabase
      .from('zvit_table')
      .select('organization, date_opened')
      .gte('date_opened', `${selectedYear}-01-01`)
      .lte('date_opened', `${selectedYear}-12-31`)

    if (error) {
      setReport([])
      setLoading(false)
      return
    }

    // Групуємо по організаціях і місяцях
    const orgMap: Record<string, number[]> = {}
    for (const row of data || []) {
      const org = row.organization || '—'
      const date = new Date(row.date_opened)
      if (!orgMap[org]) orgMap[org] = Array(12).fill(0)
      if (!isNaN(date.getTime()) && date.getFullYear() === selectedYear) {
        orgMap[org][date.getMonth()]++
      }
    }
    const result: ReportRow[] = Object.entries(orgMap).map(([organization, months]) => {
      const row: ReportRow = { organization }
      monthNames.forEach((m, i) => (row[m] = months[i]))
      return row
    })
    setReport(result)
    setLoading(false)
  }

  const handlePrevYear = () => setYear(y => y - 1)
  const handleNextYear = () => setYear(y => y + 1)

  // Підрахунок підсумків по місяцях (рядок) і по організаціях (стовпець)
  const monthTotals = monthNames.map(
    (month, i) => report.reduce((sum, row) => sum + (typeof row[month] === 'number' ? (row[month] as number) : 0), 0)
  )
  const orgTotals = report.map(row => monthNames.reduce((sum, month) => sum + (typeof row[month] === 'number' ? (row[month] as number) : 0), 0))
  const grandTotal = monthTotals.reduce((a, b) => a + b, 0)

  const [accountStatusSummary, setAccountStatusSummary] = useState<Record<string, { active: number; pending: number }>>({});

  useEffect(() => {
    async function fetchAccountStatus() {
      const { data, error } = await supabase
        .from('zvit_table')
        .select('organization, account_status');

      if (error) {
        console.error('Помилка отримання статусів рахунків:', error);
        return;
      }

      const statusMap: Record<string, { active: number; pending: number }> = {};

      (data || []).forEach(row => {
        const org = row.organization || '—';
        const status = row.account_status;

        if (!statusMap[org]) {
          statusMap[org] = { active: 0, pending: 0 };
        }

        if (status === 'Активний') {
          statusMap[org].active++;
        } else if (status === 'Очікує активацію') {
          statusMap[org].pending++;
        }
      });

      setAccountStatusSummary(statusMap);
    }

    fetchAccountStatus();
  }, []);

  return (
    <div className="min-h-screen bg-white text-gray-900 flex flex-col">
      <header className="bg-blue-700 text-white shadow-md">
        <div className="max-w-7xl mx-auto flex justify-between items-center px-6 py-4">
          <h1 className="text-xl font-bold">Звітна система</h1>
          <nav className="flex gap-6">
            <Link href="/" className="hover:underline text-lg">Головна</Link>
            <Link href="/zvit" className="hover:underline text-lg">Звіт</Link>
            <Link href="/admin" className="hover:underline text-lg">Адмін-панель</Link>
          </nav>
        </div>
      </header>

      <div className="w-full max-w-none mx-auto px-2 sm:px-4">
        <div className="flex items-center justify-between mb-6">
          <button onClick={handlePrevYear} className="px-4 py-2 bg-gray-200 rounded hover:bg-gray-300 text-lg font-bold">←</button>
          <h2 className="text-2xl md:text-3xl font-extrabold text-center tracking-tight">Звіт по відкриттю рахунків за {year} рік</h2>
          <button onClick={handleNextYear} className="px-4 py-2 bg-gray-200 rounded hover:bg-gray-300 text-lg font-bold">→</button>
        </div>
        {/* Основний звіт: Організація × Місяць (відкриті рахунки) */}
        <div className="mb-10 w-full px-4">
          <h2 className="text-xl font-bold mb-4">Звіт по відкриттю рахунків</h2>
          <table className="table-auto w-full border-collapse border border-gray-300">
            <thead>
              <tr>
                <th className="border border-gray-300 px-4 py-2 text-left">Організація</th>
                {monthNames.map(month => (
                  <th key={month} className="border border-gray-300 px-4 py-2 text-center">{month}</th>
                ))}
                <th className="border border-gray-300 px-4 py-2 text-center">Разом</th>
              </tr>
            </thead>
            <tbody>
              {report.map((row, idx) => (
                <tr key={idx}>
                  <td className="border border-gray-300 px-4 py-2 font-semibold whitespace-nowrap">{row.organization}</td>
                  {monthNames.map((month, i) => (
                    <td key={i} className="border border-gray-300 px-4 py-2 text-center">{row[month]}</td>
                  ))}
                  <td className="border border-gray-300 px-4 py-2 text-center font-bold">{orgTotals[idx]}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td className="border border-gray-300 px-4 py-2 font-bold">Разом</td>
                {monthTotals.map((total, i) => (
                  <td key={i} className="border border-gray-300 px-4 py-2 text-center font-bold">{total}</td>
                ))}
                <td className="border border-gray-300 px-4 py-2 text-center font-extrabold">{grandTotal}</td>
              </tr>
            </tfoot>
          </table>
        </div>

        {/* Додатковий звіт: Організація × Місяць (активовані рахунки) */}
        <div className="mb-10 w-full px-4">
          <h2 className="text-xl font-bold mb-4">Активовані рахунки по організаціях</h2>
          <table className="table-auto w-full border-collapse border border-gray-300">
            <thead>
              <tr>
                <th className="border border-gray-300 px-4 py-2 text-left">Організація</th>
                {monthNames.map(month => (
                  <th key={month} className="border border-gray-300 px-4 py-2 text-center">{month}</th>
                ))}
                <th className="border border-gray-300 px-4 py-2 text-center">Разом</th>
              </tr>
            </thead>
            <tbody>
              {activated.orgs.map((org, idx) => (
                <tr key={idx}>
                  <td className="border border-gray-300 px-4 py-2 font-semibold whitespace-nowrap">{org}</td>
                  {monthNames.map((_, i) => (
                    <td key={i} className="border border-gray-300 px-4 py-2 text-center">{activated.orgMonthMap[org][i]}</td>
                  ))}
                  <td className="border border-gray-300 px-4 py-2 text-center font-bold">
                    {activated.orgMonthMap[org].reduce((sum, count) => sum + count, 0)}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td className="border border-gray-300 px-4 py-2 font-bold">Разом</td>
                {monthNames.map((_, i) => (
                  <td key={i} className="border border-gray-300 px-4 py-2 text-center font-bold">
                    {activated.orgs.reduce((sum, org) => sum + activated.orgMonthMap[org][i], 0)}
                  </td>
                ))}
                <td className="border border-gray-300 px-4 py-2 text-center font-extrabold">
                  {activated.orgs.reduce((total, org) => total + activated.orgMonthMap[org].reduce((sum, count) => sum + count, 0), 0)}
                </td>
              </tr>
            </tfoot>
          </table>
        </div>

        {/* Підсумкова таблиця: Організація × Активні × Очікує активацію */}
        <div className="mb-10 px-4">
          <h2 className="text-xl font-bold mb-4">Підсумок по організаціях</h2>
          <table className="table-auto border-collapse border border-gray-300">
            <thead>
              <tr>
                <th className="border border-gray-300 px-4 py-2 text-left">Назва організації</th>
                <th className="border border-gray-300 px-4 py-2 text-center">Активні</th>
                <th className="border border-gray-300 px-4 py-2 text-center">Очікує активацію</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(accountStatusSummary).map(([org, summary], idx) => (
                <tr key={idx}>
                  <td className="border border-gray-300 px-4 py-2 font-semibold whitespace-nowrap">{org}</td>
                  <td className="border border-gray-300 px-4 py-2 text-center">{summary.active}</td>
                  <td className="border border-gray-300 px-4 py-2 text-center">{summary.pending}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
