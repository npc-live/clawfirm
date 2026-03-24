# whips/gaokao

高考志愿填报 AI 辅助决策系统。综合分数位次、地域偏好、家庭经济、职业目标，生成完整的院校+专业填报方案。

## 工作流文件

| 文件 | 职责 |
|------|------|
| `run-all.whip` | **一键全流程**：对话收集信息 → 自动顺序执行所有阶段 |
| `setup.whip` | 建立学生画像、分数解读、填报策略框架 |
| `research.whip` | 院校调研（历年录取线）、专业就业分析、城市生活成本 |
| `match.whip` | 多维度评分匹配，生成冲/稳/保候选列表 |
| `plan.whip` | 确定最终志愿顺序，生成可直接填写的方案 |
| `report.whip` | 综合报告，适合学生和家长共同阅读 |

## 使用流程

### 一键运行（推荐）

```bash
whipflow run whips/gaokao/run-all.whip
```

系统会依次问你 4 个问题，填完后自动跑完所有阶段，最终输出完整报告。

**系统会问的 4 个问题：**

1. 在哪个省份参加高考？
2. 预估或实际高考分数？（只填数字）
3. 性别？（男 / 女）
4. 家庭经济情况？（随便说，如：普通工薪 / 农村家庭比较困难 / 父母做生意比较宽裕）

第 5 问（可选）：有没有其他补充，比如想学什么专业、不想去哪些城市、有没有特别的职业目标。直接回车跳过也行。

---

### 分步运行（高级用法）

先创建配置文件（只需 4 个核心字段）：

```bash
cat > data/current-run.json << 'EOF'
{
  "province": "广东",
  "score": 612,
  "gender": "男",
  "family": "普通，父母工薪，家里还有一个弟弟在读初中"
}
EOF
```

`family` 字段直接用自然语言描述，AI 会自动推断经济水平和策略。

可选补充字段（填了更准确，不填 AI 会自动推断）：

```json
{
  "province": "广东",
  "score": 612,
  "gender": "男",
  "family": "普通，父母工薪",
  "extra": "喜欢计算机，不想去东北，目标互联网大厂"
}
```

然后逐步执行：

```bash
# 1. 建立画像 + 生成策略
whipflow run whips/gaokao/setup.whip
# → docs/gaokao-strategy.md    总体策略
# → data/gaokao-profile.json   完整学生画像

# 2. 院校与专业调研
whipflow run whips/gaokao/research.whip
# → docs/school-research.md    候选院校详细数据（15-20 所）
# → docs/major-research.md     专业就业分析
# → docs/city-research.md      目标城市生活成本对比

# 3. 智能匹配评分
whipflow run whips/gaokao/match.whip
# → docs/match-analysis.md     多维评分排行 + 风险提示
# → data/gaokao-matches.json   评分数据

# 4. 生成最终方案
whipflow run whips/gaokao/plan.whip
# → docs/final-plan.md         可直接填写的志愿清单 + 打印版速查表
# → data/gaokao-plan.json      方案数据

# 5. 生成完整报告（给家长看）
whipflow run whips/gaokao/report.whip
# → docs/gaokao-final-report.md  综合决策报告（中文通俗版）
```

---

## 典型场景

### 场景 1：高分考生，目标顶尖高校

```json
{ "province": "湖北", "score": 680, "gender": "男", "family": "家庭条件好，支持去外地" }
```
AI 推断：愿意去一线城市 → 985 冲刺 CS/AI 专业，关注清北录取线波动

### 场景 2：中等分数，家庭困难，需要奖学金

```json
{ "province": "河南", "score": 530, "gender": "女", "family": "农村家庭，父母务农，经济困难" }
```
AI 推断：奖学金是硬需求，避开高消费城市 → 省属重点 + 奖学金政策好的院校

### 场景 3：不愿离家，目标稳定职业

```json
{
  "province": "四川", "score": 575, "gender": "女",
  "family": "普通工薪家庭",
  "extra": "最好留四川，最远去重庆，考虑以后考公或者进医院"
}
```
AI 推断：省内医学/师范/行政类 → 川大华西、川北医学院优先

---

## 生成文件说明

```
docs/
├── gaokao-strategy.md          总体填报策略（城市/专业优先级）
├── school-research.md          候选院校详细数据（15-20 所）
├── major-research.md           专业就业分析（薪资/趋势/坑点）
├── city-research.md            目标城市生活成本对比
├── match-analysis.md           多维评分候选列表 + 风险提示
├── final-plan.md               最终志愿方案 + 打印版速查表
└── gaokao-final-report.md      综合决策报告（通俗版，适合家长）

data/
├── current-run.json            输入配置（4 个必填字段）
├── gaokao-profile.json         学生完整画像（AI 推断生成）
├── gaokao-candidates.json      候选院校数据库
├── gaokao-matches.json         评分排序后的候选列表
└── gaokao-plan.json            最终确认的志愿方案
```

---

## 免责声明

本系统基于 AI 推理，院校录取线数据为基于历史规律的**估算**，不构成录取承诺。
实际录取线每年因报名人数、题目难度等因素有所波动。

**强烈建议同时参考：**
- 所在省份考试院官方发布的历年录取数据
- 目标院校官方招生章程（当年版本）
- 在招生咨询日直接联系招生老师确认
