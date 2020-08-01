import React from 'react';

export type UserData = {
  id: string;
  orgRole: string;
  teamRoles?: Map<string, string> | null;
  projectRoles?: Map<string, string> | null;
};

type UserContextState = {
  user: UserData | null;
  setUser: (user: UserData | null) => void;
};
export const UserContext = React.createContext<UserContextState>({ user: null, setUser: _user => null });

export default UserContext;
