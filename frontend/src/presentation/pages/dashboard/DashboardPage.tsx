import * as React from 'react'
import { Users, DollarSign, CreditCard, Activity, Plus } from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'

import { useAppSelector } from '@/application/hooks/useAppSelector'
import { getUserFromToken } from '@/utils/auth'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  StatsCard,
  StatsCardSkeleton,
  DataTable,
  DataTableColumnHeader,
  EmptyState
} from '@/presentation/components/dashboard'

// Demo data types
interface RecentSale {
  id: string
  name: string
  email: string
  amount: string
  status: 'pending' | 'completed' | 'failed'
}

// Demo data
const recentSales: RecentSale[] = [
  {
    id: '1',
    name: 'Olivia Martin',
    email: 'olivia@example.com',
    amount: '$1,999.00',
    status: 'completed'
  },
  {
    id: '2',
    name: 'Jackson Lee',
    email: 'jackson@example.com',
    amount: '$39.00',
    status: 'completed'
  },
  {
    id: '3',
    name: 'Isabella Nguyen',
    email: 'isabella@example.com',
    amount: '$299.00',
    status: 'pending'
  },
  {
    id: '4',
    name: 'William Kim',
    email: 'william@example.com',
    amount: '$99.00',
    status: 'completed'
  },
  {
    id: '5',
    name: 'Sofia Davis',
    email: 'sofia@example.com',
    amount: '$39.00',
    status: 'failed'
  }
]

const columns: ColumnDef<RecentSale>[] = [
  {
    accessorKey: 'name',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    )
  },
  {
    accessorKey: 'email',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Email" />
    )
  },
  {
    accessorKey: 'amount',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Amount" />
    )
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.getValue('status') as RecentSale['status']
      const variants = {
        completed: 'success',
        pending: 'warning',
        failed: 'destructive'
      } as const

      return <Badge variant={variants[status]}>{status}</Badge>
    }
  }
]

export function DashboardPage() {
  const { email } = useAppSelector((state) => state.user)
  const user = getUserFromToken()
  const [isLoading, setIsLoading] = React.useState(true)

  // Simulate loading
  React.useEffect(() => {
    const timer = setTimeout(() => setIsLoading(false), 1500)
    return () => clearTimeout(timer)
  }, [])

  const stats = [
    {
      title: 'Total Revenue',
      value: '$45,231.89',
      description: '+20.1% from last month',
      icon: DollarSign,
      trend: { value: 20.1, isPositive: true }
    },
    {
      title: 'Subscriptions',
      value: '+2,350',
      description: '+180.1% from last month',
      icon: Users,
      trend: { value: 180.1, isPositive: true }
    },
    {
      title: 'Sales',
      value: '+12,234',
      description: '+19% from last month',
      icon: CreditCard,
      trend: { value: 19, isPositive: true }
    },
    {
      title: 'Active Now',
      value: '+573',
      description: '-5% from last hour',
      icon: Activity,
      trend: { value: 5, isPositive: false }
    }
  ]

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight md:text-3xl">
            Dashboard
          </h1>
          <p className="text-sm text-muted-foreground md:text-base">
            Welcome back, {user?.name || email || 'User'}
          </p>
        </div>
        <Button size="sm" className="w-fit">
          <Plus className="mr-2 size-4" />
          New Project
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-2 gap-3 md:gap-4 lg:grid-cols-4">
        {isLoading
          ? Array.from({ length: 4 }).map((_, i) => (
              <StatsCardSkeleton key={i} />
            ))
          : stats.map((stat) => (
              <StatsCard
                key={stat.title}
                title={stat.title}
                value={stat.value}
                description={stat.description}
                icon={stat.icon}
                trend={stat.trend}
              />
            ))}
      </div>

      {/* Tabs for different views */}
      <Tabs defaultValue="overview" className="space-y-3 sm:space-y-4">
        <TabsList className="w-full justify-start overflow-x-auto">
          <TabsTrigger value="overview" className="text-xs sm:text-sm">
            Overview
          </TabsTrigger>
          <TabsTrigger value="analytics" className="text-xs sm:text-sm">
            Analytics
          </TabsTrigger>
          <TabsTrigger value="reports" className="text-xs sm:text-sm">
            Reports
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-7">
            {/* Recent Sales Table */}
            <Card className="lg:col-span-4">
              <CardHeader>
                <CardTitle>Recent Sales</CardTitle>
                <CardDescription>
                  You made 265 sales this month.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <DataTable
                  columns={columns}
                  data={recentSales}
                  searchKey="name"
                  searchPlaceholder="Search..."
                />
              </CardContent>
            </Card>

            {/* Account Info */}
            <Card className="lg:col-span-3">
              <CardHeader>
                <CardTitle>Account Info</CardTitle>
                <CardDescription>
                  Your profile and account details
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3 sm:space-y-4">
                  <div className="space-y-0.5 sm:space-y-1">
                    <p className="text-xs font-medium leading-none sm:text-sm">
                      Email
                    </p>
                    <p className="truncate text-xs text-muted-foreground sm:text-sm">
                      {email || user?.email || 'Not available'}
                    </p>
                  </div>

                  <div className="space-y-0.5 sm:space-y-1">
                    <p className="text-xs font-medium leading-none sm:text-sm">
                      Name
                    </p>
                    <p className="text-xs text-muted-foreground sm:text-sm">
                      {user?.name || 'Not available'}
                    </p>
                  </div>

                  <div className="space-y-0.5 sm:space-y-1">
                    <p className="text-xs font-medium leading-none sm:text-sm">
                      Account Type
                    </p>
                    <Badge variant="secondary" className="text-xs">
                      Premium
                    </Badge>
                  </div>
                </div>
              </CardContent>
              <CardFooter>
                <Button variant="outline" size="sm" className="w-full">
                  Edit Profile
                </Button>
              </CardFooter>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="analytics">
          <EmptyState
            title="No analytics data yet"
            description="Analytics will appear here once you have more activity on your account."
            action={{
              label: 'Learn More',
              onClick: () => console.log('Learn more clicked')
            }}
          />
        </TabsContent>

        <TabsContent value="reports">
          <EmptyState
            title="No reports available"
            description="Reports will be generated automatically based on your activity."
            action={{
              label: 'Generate Report',
              onClick: () => console.log('Generate report clicked')
            }}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}
