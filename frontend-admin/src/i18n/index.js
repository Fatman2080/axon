import { createI18n } from 'vue-i18n';
const messages = {
    zh: {
        menu: {
            dashboard: '仪表盘',
            admins: '管理员',
            users: '用户',
            agentPool: 'Agent 池',
            agentStats: 'Agent 统计',
            inviteCodes: '邀请码',
            settings: '系统配置'
        },
        common: {
            loading: '加载中...',
            save: '保存',
            saving: '保存中...',
            cancel: '取消',
            close: '关闭',
            edit: '编辑',
            delete: '删除',
            deleting: '删除中...',
            confirmDelete: '确认删除',
            search: '搜索',
            clear: '清除',
            sync: '同步',
            syncAll: '全部同步',
            syncing: '同步中...',
            refresh: '刷新',
            total: '共 {count} 条',
            empty: '暂无数据',
            actions: '操作',
            password: '密码',
            email: '邮箱',
            name: '名称',
            createdAt: '创建时间',
            page: '页'
        },
        layout: {
            title: 'OpenFi 管理后台',
            logout: '退出登录'
        },
        dashboard: {
            title: '仪表盘',
            desc: 'Agent 账户分配和用户注册概览。',
            loadError: '加载仪表盘数据失败',
            totalUsers: '总用户数',
            totalAgentAccounts: '总 Agent 账户',
            assignedAgents: '已分配 Agent',
            unusedAgents: '未使用 Agent',
            totalInviteCodes: '总邀请码',
            activeInviteCodes: '有效邀请码'
        },
        admins: {
            title: '管理员',
            desc: '管理管理员账户、重置密码和删除。',
            add: '添加管理员',
            createModalTitle: '添加管理员',
            createBtn: '创建',
            creatingBtn: '创建中...',
            resetPassword: '重置密码',
            passwordLength: '至少 6 个字符',
            emailPlaceholder: 'admin@example.com',
            namePlaceholder: '管理员名称',
            loadError: '加载管理员列表失败',
            createError: '创建管理员失败',
            createSuccess: '管理员已创建：{email}',
            resetSuccess: '密码已更新：{email}',
            resetError: '更新密码失败',
            deleteSuccess: '管理员已删除：{email}',
            deleteError: '删除管理员失败',
            confirmDeletePrompt: '确认删除管理员 {email}？',
            promptNewPassword: '为 {email} 设置新密码（至少 6 位）：'
        },
        users: {
            title: '用户',
            desc: '查看注册用户和推特绑定状态。'
        },
        agentPool: {
            title: 'Agent 账户池',
            desc: '管理未使用与已分配的 Agent 账户，导入加密密钥。',
            allStatus: '全部状态',
            unused: '未使用',
            assigned: '已分配',
            import: '导入',
            importModal: '导入加密密钥',
            importDesc: '粘贴加密的 JSON 数据导入 Agent 账户。',
            importSuccess: '已导入 {imported} 个账户',
            importResult: '已导入 {imported}，重复 {duplicates}，无效 {invalid}',
            importError: '导入账户失败',
            publicKey: '公钥',
            status: '状态',
            assignedUser: '分配用户',
            assignedAt: '分配时间',
            deleteModalTitle: '确认删除',
            deleteModalDesc: '即将删除 {count} 个 Agent 账户及其关联数据，请输入管理员密码确认。',
            deleteSuccess: '已删除 {count} 个账户',
            loadError: '加载 Agent 账户失败'
        },
        agentStats: {
            title: 'Agent 统计',
            desc: '查看已分配 Agent 账户的 Hyperliquid 交易数据。',
            accountValue: '账户价值',
            totalPnl: '总盈亏',
            lastSynced: '上次同步',
            neverSynced: '从未',
            editTitle: '编辑 Agent 资料',
            description: '描述',
            performanceFee: '管理费率 (0~1)',
            loadError: '加载 Agent 统计数据失败',
            updateError: '更新 Agent 资料失败',
            syncError: '同步失败',
            deleteModalDesc: '即将删除 {count} 个 Agent 及关联数据（快照、成交、评价），请输入管理员密码确认。'
        },
        inviteCodes: {
            title: '邀请码',
            desc: '生成与管理邀请码。',
            generate: '生成邀请码',
            generateModal: '生成邀请码',
            generateDesc: '生成指定数量的邀请码。',
            count: '生成数量 (1-100)',
            generateBtn: '生成',
            generatingBtn: '生成中...',
            loadError: '加载邀请码失败',
            generateSuccess: '成功生成 {count} 个邀请码',
            generateError: '生成邀请码失败',
            code: '代码',
            status: '状态',
            usedBy: '使用者',
            usedAt: '使用时间',
            unused: '未使用',
            used: '已使用',
            allStatus: '全部状态'
        },
        settings: {
            title: '系统配置',
            desc: '管理全局设置，如实习生名额或 Season 配置。',
            dailySlots: '每日实习生名额',
            updateSuccess: '配置更新成功',
            updateError: '更新配置失败',
            loadError: '加载配置失败'
        },
        login: {
            title: '管理后台登录',
            email: '管理邮箱',
            emailPlaceholder: 'admin@example.com',
            password: '密码',
            submit: '进入控制台',
            error: '登录失败，请检查凭据'
        }
    },
    en: {
        menu: {
            dashboard: 'Dashboard',
            admins: 'Admins',
            users: 'Users',
            agentPool: 'Agent Pool',
            agentStats: 'Agent Stats',
            inviteCodes: 'Invite Codes',
            settings: 'Settings'
        },
        common: {
            loading: 'Loading...',
            save: 'Save',
            saving: 'Saving...',
            cancel: 'Cancel',
            close: 'Close',
            edit: 'Edit',
            delete: 'Delete',
            deleting: 'Deleting...',
            confirmDelete: 'Confirm Deletion',
            search: 'Search',
            clear: 'Clear',
            sync: 'Sync',
            syncAll: 'Sync All',
            syncing: 'Syncing...',
            refresh: 'Refresh',
            total: 'Total {count}',
            empty: 'No data available',
            actions: 'Actions',
            password: 'Password',
            email: 'Email',
            name: 'Name',
            createdAt: 'Created At',
            page: 'Page'
        },
        layout: {
            title: 'OpenFi Admin Console',
            logout: 'Logout'
        },
        dashboard: {
            title: 'Dashboard',
            desc: 'Overview of Agent account allocation and user registrations.',
            loadError: 'Failed to load dashboard data',
            totalUsers: 'Total Users',
            totalAgentAccounts: 'Total Agent Accounts',
            assignedAgents: 'Assigned Agents',
            unusedAgents: 'Unused Agents',
            totalInviteCodes: 'Total Invite Codes',
            activeInviteCodes: 'Active Invite Codes'
        },
        admins: {
            title: 'Admins',
            desc: 'Manage admin accounts, reset passwords, and delete.',
            add: 'Add Admin',
            createModalTitle: 'Add Admin',
            createBtn: 'Create',
            creatingBtn: 'Creating...',
            resetPassword: 'Reset Password',
            passwordLength: 'At least 6 characters',
            emailPlaceholder: 'admin@example.com',
            namePlaceholder: 'Admin Name',
            loadError: 'Failed to load admins',
            createError: 'Failed to create admin',
            createSuccess: 'Admin created: {email}',
            resetSuccess: 'Password updated: {email}',
            resetError: 'Failed to update password',
            deleteSuccess: 'Admin deleted: {email}',
            deleteError: 'Failed to delete admin',
            confirmDeletePrompt: 'Confirm deletion of admin {email}?',
            promptNewPassword: 'Set new password for {email} (min 6 chars):'
        },
        users: {
            title: 'Users',
            desc: 'View registered users and Twitter binding status.'
        },
        agentPool: {
            title: 'Agent Account Pool',
            desc: 'Manage unused and assigned Agent accounts, import encrypted keys.',
            allStatus: 'All Status',
            unused: 'Unused',
            assigned: 'Assigned',
            import: 'Import',
            importModal: 'Import Encrypted Keys',
            importDesc: 'Paste encrypted JSON data to import Agent accounts.',
            importSuccess: 'Imported {imported} accounts',
            importResult: 'Imported {imported}, Duplicates {duplicates}, Invalid {invalid}',
            importError: 'Failed to import accounts',
            publicKey: 'Public Key',
            status: 'Status',
            assignedUser: 'Assigned To',
            assignedAt: 'Assigned At',
            deleteModalTitle: 'Confirm Deletion',
            deleteModalDesc: 'About to delete {count} Agent accounts and associated data. Please enter admin password to confirm.',
            deleteSuccess: 'Deleted {count} accounts',
            loadError: 'Failed to load Agent accounts'
        },
        agentStats: {
            title: 'Agent Stats',
            desc: 'View Hyperliquid trading data for assigned Agent accounts.',
            accountValue: 'Account Value',
            totalPnl: 'Total PnL',
            lastSynced: 'Last Synced',
            neverSynced: 'Never',
            editTitle: 'Edit Agent Profile',
            description: 'Description',
            performanceFee: 'Performance Fee (0~1)',
            loadError: 'Failed to load Agent stats',
            updateError: 'Failed to update Agent profile',
            syncError: 'Failed to sync',
            deleteModalDesc: 'About to delete {count} Agents and associated data (snapshots, fills, reviews). Please enter admin password to confirm.'
        },
        inviteCodes: {
            title: 'Invite Codes',
            desc: 'Generate and manage invite codes.',
            generate: 'Generate Codes',
            generateModal: 'Generate Invite Codes',
            generateDesc: 'Generate a specific number of invite codes.',
            count: 'Quantity (1-100)',
            generateBtn: 'Generate',
            generatingBtn: 'Generating...',
            loadError: 'Failed to load invite codes',
            generateSuccess: 'Successfully generated {count} invite codes',
            generateError: 'Failed to generate invite codes',
            code: 'Code',
            status: 'Status',
            usedBy: 'Used By',
            usedAt: 'Used At',
            unused: 'Unused',
            used: 'Used',
            allStatus: 'All Status'
        },
        settings: {
            title: 'System Settings',
            desc: 'Manage global settings, such as Intern slots or Season configs.',
            dailySlots: 'Daily Intern Slots',
            updateSuccess: 'Settings updated successfully',
            updateError: 'Failed to update settings',
            loadError: 'Failed to load settings'
        },
        login: {
            title: 'Admin Console Login',
            email: 'Admin Email',
            emailPlaceholder: 'admin@example.com',
            password: 'Password',
            submit: 'Enter Console',
            error: 'Login failed, please check your credentials'
        }
    }
};
const getBrowserLanguage = () => {
    if (typeof window === 'undefined')
        return 'zh';
    const parser = navigator.language || (navigator.languages && navigator.languages[0]) || '';
    if (parser && parser.toLowerCase().startsWith('zh')) {
        return 'zh';
    }
    return 'en';
};
const i18n = createI18n({
    legacy: false, // Use Composition API
    locale: getBrowserLanguage(),
    fallbackLocale: 'en',
    messages
});
export default i18n;
