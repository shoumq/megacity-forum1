import React, { useEffect, useState } from 'react';
import { ShoppingBag, Coins, Check, Sparkles } from 'lucide-react';
import './store.css';

const api = async (path, options = {}) => {
  const token = localStorage.getItem('megacity-token');
  const response = await fetch(`/api${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers },
  });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || 'Ошибка запроса');
  return data;
};

export { api };

export function Store({ player, onPlayer, onLogin, notify }) {
  const [products, setProducts] = useState([]);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  useEffect(() => { api('/shop/products').then(x => setProducts(x.products)).catch(e => setError(`${e.message}. Запустите Go API: go run ./backend`)); }, []);
  const buy = async product => {
    if (!player) { onLogin(); return; }
    setBusy(product.id); setError('');
    try { const result = await api('/shop/buy', { method: 'POST', body: JSON.stringify({ productId: product.id }) }); onPlayer(result.player); notify(result.message); }
    catch (e) { setError(e.message); }
    finally { setBusy(''); }
  };
  return <div className="content-page store-page">
    <div className="store-hero"><div><div className="page-eyebrow">МАГАЗИН MEGACITY</div><h1>Добавь городу красок ✨</h1><p>Уникальные вещи для твоей истории в городе.</p></div><div className="wallet"><span className="coin-icon"><Coins size={19}/></span><div><small>ТВОЙ БАЛАНС</small><b>{player?.coins?.toLocaleString('ru-RU') ?? '—'} <i>MC</i></b></div><button onClick={onLogin}>{player ? player.nickname : 'Войти'}</button></div></div>
    <div className="store-info"><span>🎁</span><p><b>Бонус новичка</b><br/>Зарегистрируйся и получи <strong>1 000 MC</strong> на покупки в магазине.</p></div>
    {error && <div className="store-error">{error}</div>}
    <div className="store-toolbar"><div><div className="page-eyebrow">ВЫБИРАЙ СВОЁ</div><h2>Витрина города <span>{products.length}</span></h2></div><small>Игровая валюта: <b>MC</b></small></div>
    <div className="products-grid">{products.map((p, i) => {
      const owned = player?.purchases?.includes(p.id);
      return <article className="product-card" key={p.id}><div className={`product-art product-art-${i % 4}`}><span>{p.icon}</span><i>✦</i></div><div className="product-details"><small>{p.category}</small><h3>{p.name}</h3><p>{p.description}</p><div className="product-buy"><b><Coins size={15}/>{p.price.toLocaleString('ru-RU')} <i>MC</i></b><button disabled={busy === p.id || owned} onClick={() => buy(p)}>{owned ? <><Check size={14}/> Куплено</> : busy === p.id ? 'Покупка…' : 'Купить'}</button></div></div></article>;
    })}</div>
    {!products.length && !error && <div className="store-loading"><Sparkles size={20}/> Загружаем витрину…</div>}
    <p className="store-disclaimer"><ShoppingBag size={14}/> Покупки сохраняются в профиле игрока. Оплата реальными деньгами в прототипе отключена.</p>
  </div>;
}
