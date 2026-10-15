import React, { createContext, useContext, useState, useEffect } from 'react';
import { User } from '../types';
import { api } from '../api/client';

interface AuthContextType {
  user: User | null;
  loading: boolean;
  setupRequired: boolean;
  login: (email: string, pass: string, totp?: string) => Promise<{ totp_required?: boolean }>;
  setupOwner: (email: string, pass: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [setupRequired, setSetupRequired] = useState(false);

  const refresh = async () => {
    try {
      const status = await api.getAuthStatus();
      setSetupRequired(status.setup_required);
      if (status.authenticated && status.user) {
        setUser(status.user);
      } else {
        setUser(null);
      }
    } catch (err) {
      console.error('Failed to load auth status', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const login = async (email: string, pass: string, totp?: string) => {
    const res = await api.login(email, pass, totp);
    if (res.totp_required) {
      return { totp_required: true };
    }
    if (res.user) {
      setUser(res.user);
    }
    return {};
  };

  const setupOwner = async (email: string, pass: string) => {
    const res = await api.setupOwner(email, pass);
    setUser(res.user);
    setSetupRequired(false);
  };

  const logout = async () => {
    await api.logout();
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, loading, setupRequired, login, setupOwner, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within an AuthProvider');
  return context;
};
