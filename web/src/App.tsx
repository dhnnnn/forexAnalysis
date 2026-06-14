import { ApolloProvider } from '@apollo/client'
import { apolloClient } from './graphql/apolloClient'
import { MainLayout } from './components/layout/MainLayout'

export default function App() {
  return (
    <ApolloProvider client={apolloClient}>
      <MainLayout />
    </ApolloProvider>
  )
}
