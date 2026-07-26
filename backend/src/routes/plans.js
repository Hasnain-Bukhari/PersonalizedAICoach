import express from 'express';
import { PrismaClient } from '@prisma/client';
const router = express.Router();
const prisma = new PrismaClient();
router.get('/plans', async (req, res) => {
    const plans = await prisma.plan.findMany({
        where: {
            userId: req.user.id,
        },
    });
    res.json(plans);
});
export default router;
//# sourceMappingURL=plans.js.map