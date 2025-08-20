"use client";

import { useState, useEffect } from 'react';
import { supabase } from '@/lib/supabase';

export default function AdminPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [userRole, setUserRole] = useState<'Менеджер' | 'Адміністратор' | null>(null);

  useEffect(() => {
    async function fetchUserRole() {
      const { data: { session } } = await supabase.auth.getSession();
      const userEmail = session?.user?.email;

      if (userEmail) {
        const { data, error } = await supabase
          .from('users')
          .select('role')
          .eq('email', userEmail)
          .single();

        if (error) {
          console.error('Помилка отримання ролі користувача:', error);
        } else {
          setUserRole(data?.role || null);
        }
      }
    }

    fetchUserRole();
  }, []);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    const { error } = await supabase.auth.signInWithPassword({
      email,
      password,
    });

    if (error) {
      setError(error.message);
    } else {
      alert('Успішний вхід!');
    }
  };

  if (userRole === 'Менеджер') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <p className="text-red-500 text-lg">У вас немає доступу до адмін-панелі.</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-6 rounded shadow-md w-full max-w-sm">
        <h1 className="text-xl font-bold mb-4">Адмін-панель</h1>
        <form onSubmit={handleLogin}>
          <div className="mb-4">
            <label htmlFor="email" className="block text-sm font-medium text-gray-700">Email</label>
            <input
              type="email"
              id="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded shadow-sm focus:outline-none focus:ring focus:ring-blue-500"
              required
            />
          </div>
          <div className="mb-4">
            <label htmlFor="password" className="block text-sm font-medium text-gray-700">Пароль</label>
            <input
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded shadow-sm focus:outline-none focus:ring focus:ring-blue-500"
              required
            />
          </div>
          {error && <p className="text-red-500 text-sm mb-4">{error}</p>}
          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 focus:outline-none focus:ring focus:ring-blue-500"
          >
            Увійти
          </button>
        </form>
      </div>
    </div>
  );
}
