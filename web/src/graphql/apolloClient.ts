import { ApolloClient, InMemoryCache, HttpLink, split } from '@apollo/client'
import { GraphQLWsLink } from '@apollo/client/link/subscriptions'
import { createClient } from 'graphql-ws'
import { getMainDefinition } from '@apollo/client/utilities'

const GRAPHQL_HTTP = import.meta.env.VITE_GRAPHQL_HTTP ?? '/graphql'
const GRAPHQL_WS   = import.meta.env.VITE_GRAPHQL_WS  ?? 
  (window.location.protocol === 'https:' ? 'wss://' : 'ws://') +
  window.location.host + '/graphql'

const httpLink = new HttpLink({
  uri: GRAPHQL_HTTP,
})

const wsLink = new GraphQLWsLink(
  createClient({
    url: GRAPHQL_WS,
    retryAttempts: Infinity,
    shouldRetry: () => true,
    retryWait: async (retries) => {
      // Exponential backoff: 1s, 2s, 4s, 8s, max 30s
      const delay = Math.min(1000 * Math.pow(2, retries), 30_000)
      await new Promise((resolve) => setTimeout(resolve, delay))
    },
    on: {
      connected: () => {
        console.info('[WS] GraphQL WebSocket connected')
      },
      closed: () => {
        console.warn('[WS] GraphQL WebSocket disconnected')
      },
      error: (err) => {
        console.error('[WS] GraphQL WebSocket error', err)
      },
    },
  })
)

// Route subscriptions to WS, queries/mutations to HTTP
const splitLink = split(
  ({ query }) => {
    const definition = getMainDefinition(query)
    return (
      definition.kind === 'OperationDefinition' &&
      definition.operation === 'subscription'
    )
  },
  wsLink,
  httpLink
)

export const apolloClient = new ApolloClient({
  link: splitLink,
  cache: new InMemoryCache({
    typePolicies: {
      Candle: { keyFields: ['pair', 'timestamp', 'timeframe'] },
      AgentDebateEntry: { keyFields: ['id'] },
      SignalEntry: { keyFields: ['id'] },
      KnowledgeRule: { keyFields: ['id'] },
    },
  }),
  defaultOptions: {
    watchQuery: { fetchPolicy: 'cache-and-network' },
  },
})
