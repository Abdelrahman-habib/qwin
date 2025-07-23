import { createContext, useState, ReactNode } from "react";

interface TitleContextType {
  title: string;
  setTitle: (title: string) => void;
}

export const TitleContext = createContext<TitleContextType | undefined>(
  undefined
);

interface TitleProviderProps {
  children: ReactNode;
}

export function TitleProvider({ children }: TitleProviderProps) {
  const [title, setTitle] = useState("qwin");

  return (
    <TitleContext.Provider value={{ title, setTitle }}>
      {children}
    </TitleContext.Provider>
  );
}
