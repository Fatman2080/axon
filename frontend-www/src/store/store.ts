import { configureStore } from '@reduxjs/toolkit';
import strategyReducer from './slices/strategySlice';
import agentReducer from './slices/agentSlice';
import userReducer from './slices/userSlice';

export const store = configureStore({
  reducer: {
    strategies: strategyReducer,
    agents: agentReducer,
    user: userReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
