import React, { useEffect } from 'react';
import { X } from 'lucide-react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  maxWidth?: string;
}

export const Modal: React.FC<ModalProps> = ({ isOpen, onClose, title, children, maxWidth = 'max-w-xl' }) => {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    if (isOpen) {
      document.body.style.overflow = 'hidden';
      window.addEventListener('keydown', handleKeyDown);
    }
    return () => {
      document.body.style.overflow = 'unset';
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/30 backdrop-blur-xs animate-fade-in">
      <div
        className={`w-full ${maxWidth} bg-white border border-zinc-200 rounded-xl shadow-xl overflow-hidden flex flex-col max-h-[90vh]`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-6 py-4 border-b border-zinc-100 bg-white">
          <h3 className="text-base font-semibold text-zinc-900 tracking-tight">{title}</h3>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-zinc-400 hover:text-black hover:bg-zinc-100 transition"
            aria-label="Close modal"
          >
            <X size={18} />
          </button>
        </div>
        <div className="p-6 overflow-y-auto text-zinc-900">{children}</div>
      </div>
    </div>
  );
};
