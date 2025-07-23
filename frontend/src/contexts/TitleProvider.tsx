import { useState, ReactNode } from "react";
import { TitleContext } from "./TitleContext";

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
